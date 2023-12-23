import './commands';

beforeEach(() => {
  cy.intercept('GET', '/sockjs-node/**', (req) => {
    req.destroy();
  });

  cy.intercept('GET', '/api/health/**', (req) => {
    req.reply({});
  });
});
