module.exports = {
  publicPath: process.env.LISTMONK_FRONTEND_ROOT || '/admin',
  outputDir: 'dist',

  

  // This is to make all static file requests generated by Vue to go to
  // /frontend/*. However, this also ends up creating a `dist/frontend`
  // directory and moves all the static files in it. The physical directory
  // and the URI for assets are tightly coupled. This is handled in the Go app
  // by using stuffbin aliases.
  assetsDir: 'static',

  // Move the index.html file from dist/index.html to dist/frontend/index.html
  // indexPath: './frontend/index.html',

  productionSourceMap: false,
  filenameHashing: true,

  css: {
    loaderOptions: {
      sass: {
        implementation: require('sass'), // This line must in sass option
      },
    },
  },

  devServer: {
    watchOptions: {
      aggregateTimeout: 200,
      poll: 1000,
    },
    port: process.env.LISTMONK_FRONTEND_PORT || 8080,
    proxy: {
      '^/$': {
        target: process.env.LISTMONK_API_URL || 'http://127.0.0.1:9000'
      },
      '^/(api|webhooks|subscription|public|health)': {
        target: process.env.LISTMONK_API_URL || 'http://127.0.0.1:9000'
      },
      '^/(admin\/custom\.(css|js))': {
        target: process.env.LISTMONK_API_URL || 'http://127.0.0.1:9000'
      }
    }
  }
};
