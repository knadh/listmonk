import './commands';

beforeEach(() => {
  cy.server({
    ignore: (xhr) => {
      // Ignore the webpack dev server calls that interfere in the tests
      // when testing with `yarn serve`.
      if (xhr.url.indexOf('sockjs-node/') > -1) {
        return true;
      }

      if (xhr.url.indexOf('api/health') > -1) {
        return true;
      }

      // Return the default cypress whitelist filer.
      return xhr.method === 'GET' && /\.(jsx?|html|css)(\?.*)?$/.test(xhr.url);
    },
  });

  cy.intercept('GET', '/api/health', {});
});
