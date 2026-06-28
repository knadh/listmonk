const { defineConfig } = require('cypress');

module.exports = defineConfig({
  env: {
    apiUrl: 'http://localhost:9000',
    serverInitCmd:
      'pkill -9 listmonk; cd ../ && LISTMONK_ADMIN_USER=admin LISTMONK_ADMIN_PASSWORD=listmonk ./listmonk --install --yes && setsid ./listmonk </dev/null >/dev/null 2>&1 &',
    serverInitBlankCmd:
      'pkill -9 listmonk; cd ../ && ./listmonk --install --yes && setsid ./listmonk </dev/null >/dev/null 2>&1 &',
    LISTMONK_ADMIN_USER: 'admin',
    LISTMONK_ADMIN_PASSWORD: 'listmonk',
  },
  viewportWidth: 1400,
  viewportHeight: 950,
  e2e: {
    experimentalRunAllSpecs: true,
    testIsolation: false,
    experimentalSessionAndOrigin: false,
    // We've imported your old cypress plugins here.
    // You may want to clean this up later by importing these.
    setupNodeEvents(on, config) {
      return require('./cypress/plugins/index.js')(on, config);
    },
    baseUrl: 'http://localhost:9000',
  },
});
