const { defineConfig } = require('cypress')

module.exports = defineConfig({
  env: {
    apiUrl: 'http://localhost:9000',
    serverInitCmd:
      'pkill -9 listmonk | cd ../ && ./listmonk --install --yes && ./listmonk > /dev/null 2>/dev/null &',
    username: 'listmonk',
    password: 'listmonk',
  },
  viewportWidth: 1300,
  viewportHeight: 950,
  e2e: {
    // We've imported your old cypress plugins here.
    // You may want to clean this up later by importing these.
    setupNodeEvents(on, config) {
      return require('./cypress/plugins/index.js')(on, config)
    },
    baseUrl: 'http://localhost:9000/admin',
  },
})
