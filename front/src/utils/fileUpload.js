import SparkMD5 from "spark-md5";

export class FileUploadUtils {
    constructor(chunkSize = 5 * 1024 * 1024) {
        this.CHUNK_SIZE = chunkSize // 分片大小
    }

    /**
     * 计算文件hash
     * @param {File} file
     * @returns
     */
    async calculateFileHash(file, chunkSize) {
        return new Promise((resolve, reject) => {
            const spark = new SparkMD5.ArrayBuffer()
            const fileReader = new FileReader()

            // 计算分片数量
            const chunks = Math.ceil(file.size / chunkSize)
            let currentChunk = 0

            // 读取文件
            fileReader.onload = (e) => {
                // 计算hash
                spark.append(e.target.result)
                currentChunk++
                if (currentChunk < chunks) {
                    loadNext()
                } else {
                    // 返回hash
                    resolve(spark.end())
                }
            }

            // 读取失败
            fileReader.onerror = (e) => {
                console.error('文件读取失败', e)
                reject(e)
            }

            // 读取下一个分片
            const loadNext = () => {
                // 计算分片开始和结束位置
                const start = currentChunk * chunkSize
                const end = Math.min(start + chunkSize, file.size)
                // 读取分片
                const chunk = file.slice(start, end)
                fileReader.readAsArrayBuffer(chunk)
            }
            loadNext()
        })
    }

    /**
     * 创建文件分片
     * @param {File} file
     * @returns
     */
    createFileChunks(file,chunkSize) {
        const chunks = []
        // 计算分片数量
        const chunkCount = Math.ceil(file.size / chunkSize)
        // 遍历分片
        for (let i = 0; i < chunkCount; i++) {
            // 计算分片开始和结束位置
            const start = i * chunkSize
            const end = Math.min(start + chunkSize, file.size)
            // 读取分片
            const chunk = file.slice(start, end)
            // 添加到chunks
            chunks.push({
                chunk,
                index: i,
                start,
                end
            })
        }
        // 返回chunks
        return chunks
    }

    // 动态计算分片大小
    calculateDynamicChunkSize(fileSize) {
        const MB = 1024 * 1024;
        const minChunkSize = 512 * 1024; // 最小分片大小
        const maxChunkSize = 5 * MB;     // 最大分片大小
        const step = 256 * 1024;         // 步进：256KB

        // 默认切 50 块
        let idealChunkSize = fileSize / 50;

        // 限制在允许范围内
        idealChunkSize = Math.max(minChunkSize, Math.min(idealChunkSize, maxChunkSize));

        return Math.ceil(idealChunkSize / step) * step;
    }

    /**
     * 格式化文件大小
     * @param {number} bytes
     * @returns
     */
    formatFileSize(bytes) {
        if (bytes === 0) return '0 Bytes'
        const k = 1024
        const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB']
        // 计算单位,这一步也就是找出 bytes 是 k 的多少次方
        const i = Math.floor(Math.log(bytes) / Math.log(k))
        // 返回格式化后的文件大小
        return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
    }

    /**
     * 检查文件类型
     * @param {File} file
     * @param {string[]} allowedTypes
     * @returns
     */
    checkFileType(file, allowedTypes = []) {
        if (allowedTypes.length === 0) return true
        let fileExtension = this.getFileExtension(file.name)
        return allowedTypes.includes(fileExtension)
    }

    /**
     * 获取文件扩展名
     * @param {*} filename
     * @returns
     */
    getFileExtension(filename) {
        if (!filename || typeof filename !== 'string') return ''
        const lastDotIndex = filename.lastIndexOf('.')
        if (lastDotIndex <= 0 || lastDotIndex === filename.length - 1) {
            return ''
        }
        return filename.slice(lastDotIndex + 1)
    }

    /**
     * 检查文件大小
     * @param {File} file
     * @param {number} maxSize
     * @returns
     */
    checkFileSize(file, maxSize = 10 * 1024 * 1024) {
        return file.size <= maxSize
    }
}