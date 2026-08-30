const { defineConfig } = require('@vue/cli-service')
const AutoImport = require('unplugin-auto-import/webpack')
const Components = require('unplugin-vue-components/webpack')
const { ElementPlusResolver } = require('unplugin-vue-components/resolvers')

module.exports = defineConfig({
  devServer: {
    proxy: {
      '/api': {
        target: process.env.VUE_APP_DEV_API_TARGET || 'http://localhost:8081',
        changeOrigin: true,
        pathRewrite: { '^/api': '' }
      }
    }
  },
  configureWebpack: {
    optimization: {
      runtimeChunk: 'single',
      splitChunks: {
        chunks: 'all',
        cacheGroups: {
          vue: {
            name: 'chunk-vue',
            test: /[\\/]node_modules[\\/](vue|vue-router|vuex)[\\/]/,
            priority: 40,
            reuseExistingChunk: true
          },
          elementPlus: {
            name: 'chunk-element-plus',
            test: /[\\/]node_modules[\\/](element-plus)[\\/]/,
            priority: 30,
            reuseExistingChunk: true
          },
          elementPlusIcons: {
            name: 'chunk-element-plus-icons',
            test: /[\\/]node_modules[\\/]@element-plus[\\/]icons-vue[\\/]/,
            priority: 35,
            reuseExistingChunk: true
          },
          markdown: {
            name: 'chunk-marked',
            test: /[\\/]node_modules[\\/](marked)[\\/]/,
            priority: 20,
            reuseExistingChunk: true
          }
        }
      }
    },
    plugins: [
      AutoImport({
        resolvers: [ElementPlusResolver()],
      }),
      Components({
        resolvers: [ElementPlusResolver()],
      }),
    ],
  },
  css: {
    loaderOptions: {
      scss: {
        additionalData: `@import "~@/styles/variables.scss";`
      }
    }
  }
})
