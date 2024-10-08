const { defineConfig } = require('@vue/cli-service')
module.exports = defineConfig({
  pwa: {
    iconPaths: {
      favicon32: 'favicon.ico',
      favicon16: 'favicon.ico',
      favicon: 'favicon.ico',
      appleTouchIcon: 'appleIcon.png',
      maskIcon: 'favicon.ico',
      msTileImage: 'favicon.ico'
    }
  },
  transpileDependencies: true,
  //sed_tag/publicPath: "/dist", //加上这一行即可
  productionSourceMap: false, // 生产环境不产生未加密的map文件
  devServer: {
    port: 4001,
    proxy: {
      '/api': {
        ws: false,
        target: process.env.VUE_APP_BASE_URL,
        changeOrigin: true
      }
    }
  },
})
