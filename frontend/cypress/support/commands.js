import 'cypress-file-upload';
import 'cypress-wait-until';

Cypress.Commands.add('resetDB', () => {
  // Although cypress clearly states that a webserver should not be run
  // from within it, listmonk is killed, the DB reset, and run again
  // in the background. If the DB is reset without restartin listmonk,
  // the live Postgres connections in the app throw errors because the
  // schema changes midway.
  cy.exec(Cypress.env('serverInitCmd'));
});

Cypress.Commands.add('resetDBBlank', () => {
  cy.exec(Cypress.env('serverInitBlankCmd'));
});

// Takes a th class selector of a Buefy table, clicks it sorting the table,
// then compares the values of [td.data-id] attri of all the rows in the
// table against the given IDs, asserting the expected order of sort.
Cypress.Commands.add('sortTable', (theadSelector, ordIDs) => {
  cy.get(theadSelector).click();
  cy.wait(250);
  cy.get('tbody td[data-id]').each(($el, index) => {
    expect(ordIDs[index]).to.equal(parseInt($el.attr('data-id')));
  });
});

Cypress.Commands.add('loginAndVisit', (url) => {
  cy.visit(`/admin/login?next=${url}`);

  const username = Cypress.env('LISTMONK_ADMIN_USER') || 'admin';
  const password = Cypress.env('LISTMONK_ADMIN_PASSWORD') || 'listmonk';

  // Fill the username and passowrd and login.
  cy.get('input[name=username]').invoke('val', username);
  cy.get('input[name=password]').invoke('val', password);

  // Submit form.
  cy.get('button').click();
});

Cypress.Commands.add('clickMenu', (...selectors) => {
  selectors.forEach((s) => {
    cy.get(`.menu a[data-cy="${s}"]`).click();
  });
});

// https://www.nicknish.co/blog/cypress-targeting-elements-inside-iframes
Cypress.Commands.add('iframe', { prevSubject: 'element' }, ($iframe, callback = () => { }) => cy
  .wrap($iframe)
  .should((iframe) => expect(iframe.contents().find('body')).to.exist)
  .then((iframe) => cy.wrap(iframe.contents().find('body')))
  .within({}, callback));

Cypress.Commands.add('waitForBackend', () => {
  // Silence all network errors during wait
  cy.intercept('*', { continueOnNetworkError: true });

  // Keep trying until backend responds successfully
  cy.waitUntil(
    () => cy.request({
      url: '/api/health',
      failOnStatusCode: false,
    }).then((res) =>
      // Re-enable exception handling once backend is healthy
      res.status === 200),
    {
      timeout: 60000,
      interval: 2000,
    },
  );
});

Cypress.on('uncaught:exception', (err, runnable) => {
  if (err.hasOwnProperty('request')) {
    return false;
  }

  return true;
});
