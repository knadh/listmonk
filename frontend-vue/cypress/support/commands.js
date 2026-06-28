import 'cypress-file-upload';
import 'cypress-wait-until';

Cypress.Commands.add('resetDB', () => {
  // Although cypress clearly states that a webserver should not be run
  // from within it, listmonk is killed, the DB reset, and run again
  // in the background. If the DB is reset without restarting listmonk,
  // the live Postgres connections in the app throw errors because the
  // schema changes midway.
  cy.task('resetServer');
  cy.waitForBackend();
});

Cypress.Commands.add('resetDBBlank', () => {
  cy.task('resetServer', { blank: true });
  cy.waitForBackend();
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
  // The server restarts after a 500ms delay on settings change.
  // Wait for the server to go down
  cy.wait(1000);

  // Keep polling the public /health endpoint until the (new) server
  // is live. Use fetch() as cy.request() throws on ECONNREFUSED even with failOnStatusCode:false.
  cy.waitUntil(
    () => cy.wrap(null, { log: false }).then(() => fetch('/health')
      .then((res) => res.status === 200)
      .catch(() => false)),
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
