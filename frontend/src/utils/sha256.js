const K = [
  0x428a2f98, 0x71374491, 0xb5c0fbcf, 0xe9b5dba5, 0x3956c25b, 0x59f111f1, 0x923f82a4, 0xab1c5ed5,
  0xd807aa98, 0x12835b01, 0x243185be, 0x550c7dc3, 0x72be5d74, 0x80deb1fe, 0x9bdc06a7, 0xc19bf174,
  0xe49b69c1, 0xefbe4786, 0x0fc19dc6, 0x240ca1cc, 0x2de92c6f, 0x4a7484aa, 0x5cb0a9dc, 0x76f988da,
  0x983e5152, 0xa831c66d, 0xb00327c8, 0xbf597fc7, 0xc6e00bf3, 0xd5a79147, 0x06ca6351, 0x14292967,
  0x27b70a85, 0x2e1b2138, 0x4d2c6dfc, 0x53380d13, 0x650a7354, 0x766a0abb, 0x81c2c92e, 0x92722c85,
  0xa2bfe8a1, 0xa81a664b, 0xc24b8b70, 0xc76c51a3, 0xd192e819, 0xd6990624, 0xf40e3585, 0x106aa070,
  0x19a4c116, 0x1e376c08, 0x2748774c, 0x34b0bcb5, 0x391c0cb3, 0x4ed8aa4a, 0x5b9cca4f, 0x682e6ff3,
  0x748f82ee, 0x78a5636f, 0x84c87814, 0x8cc70208, 0x90befffa, 0xa4506ceb, 0xbef9a3f7, 0xc67178f2
]

const H0 = [
  0x6a09e667, 0xbb67ae85, 0x3c6ef372, 0xa54ff53a,
  0x510e527f, 0x9b05688c, 0x1f83d9ab, 0x5be0cd19
]

function rotr(value, shift) {
  return (value >>> shift) | (value << (32 - shift))
}

function toHex(value) {
  return value.toString(16).padStart(8, '0')
}

class SHA256 {
  constructor() {
    this.state = H0.slice()
    this.buffer = new Uint8Array(64)
    this.bufferLength = 0
    this.bytesHashed = 0
    this.temp = new Uint32Array(64)
    this.finished = false
  }

  update(data) {
    if (this.finished) throw new Error('SHA256 digest already called')
    const bytes = data instanceof Uint8Array ? data : new Uint8Array(data)
    let position = 0
    this.bytesHashed += bytes.length

    while (position < bytes.length) {
      const take = Math.min(bytes.length - position, 64 - this.bufferLength)
      this.buffer.set(bytes.subarray(position, position + take), this.bufferLength)
      this.bufferLength += take
      position += take

      if (this.bufferLength === 64) {
        this.processBlock(this.buffer)
        this.bufferLength = 0
      }
    }
    return this
  }

  processBlock(chunk) {
    const w = this.temp
    for (let i = 0; i < 16; i++) {
      const j = i * 4
      w[i] = ((chunk[j] << 24) | (chunk[j + 1] << 16) | (chunk[j + 2] << 8) | chunk[j + 3]) >>> 0
    }
    for (let i = 16; i < 64; i++) {
      const s0 = (rotr(w[i - 15], 7) ^ rotr(w[i - 15], 18) ^ (w[i - 15] >>> 3)) >>> 0
      const s1 = (rotr(w[i - 2], 17) ^ rotr(w[i - 2], 19) ^ (w[i - 2] >>> 10)) >>> 0
      w[i] = (w[i - 16] + s0 + w[i - 7] + s1) >>> 0
    }

    let [a, b, c, d, e, f, g, h] = this.state
    for (let i = 0; i < 64; i++) {
      const s1 = (rotr(e, 6) ^ rotr(e, 11) ^ rotr(e, 25)) >>> 0
      const ch = ((e & f) ^ (~e & g)) >>> 0
      const t1 = (h + s1 + ch + K[i] + w[i]) >>> 0
      const s0 = (rotr(a, 2) ^ rotr(a, 13) ^ rotr(a, 22)) >>> 0
      const maj = ((a & b) ^ (a & c) ^ (b & c)) >>> 0
      const t2 = (s0 + maj) >>> 0
      h = g
      g = f
      f = e
      e = (d + t1) >>> 0
      d = c
      c = b
      b = a
      a = (t1 + t2) >>> 0
    }

    this.state[0] = (this.state[0] + a) >>> 0
    this.state[1] = (this.state[1] + b) >>> 0
    this.state[2] = (this.state[2] + c) >>> 0
    this.state[3] = (this.state[3] + d) >>> 0
    this.state[4] = (this.state[4] + e) >>> 0
    this.state[5] = (this.state[5] + f) >>> 0
    this.state[6] = (this.state[6] + g) >>> 0
    this.state[7] = (this.state[7] + h) >>> 0
  }

  digest() {
    if (this.finished) throw new Error('SHA256 digest already called')
    this.finished = true

    const bytesHashed = this.bytesHashed
    const left = this.bufferLength
    this.buffer[left] = 0x80
    this.buffer.fill(0, left + 1)

    if (left >= 56) {
      this.processBlock(this.buffer)
      this.buffer.fill(0)
    }

    const bitHigh = Math.floor(bytesHashed / 0x20000000)
    const bitLow = (bytesHashed << 3) >>> 0
    this.buffer[56] = (bitHigh >>> 24) & 0xff
    this.buffer[57] = (bitHigh >>> 16) & 0xff
    this.buffer[58] = (bitHigh >>> 8) & 0xff
    this.buffer[59] = bitHigh & 0xff
    this.buffer[60] = (bitLow >>> 24) & 0xff
    this.buffer[61] = (bitLow >>> 16) & 0xff
    this.buffer[62] = (bitLow >>> 8) & 0xff
    this.buffer[63] = bitLow & 0xff
    this.processBlock(this.buffer)

    return this.state.map(toHex).join('')
  }
}

export async function sha256File(file, { chunkSize = 4 * 1024 * 1024, signal, onProgress } = {}) {
  const hasher = new SHA256()
  let offset = 0

  while (offset < file.size) {
    if (signal?.aborted) throw new DOMException('Upload cancelled', 'AbortError')
    const end = Math.min(offset + chunkSize, file.size)
    const buffer = await file.slice(offset, end).arrayBuffer()
    if (signal?.aborted) throw new DOMException('Upload cancelled', 'AbortError')
    hasher.update(new Uint8Array(buffer))
    offset = end
    onProgress?.(offset / file.size)
    await new Promise(resolve => setTimeout(resolve, 0))
  }

  return hasher.digest()
}
