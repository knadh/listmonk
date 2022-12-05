const apiUrl = Cypress.env('apiUrl');

describe('Templates', () => {
  it('Opens settings page', () => {
    cy.resetDB();
    cy.loginAndVisit('/settings');
  });

  it('Changes some settings', () => {
    const rootURL = 'http://127.0.0.1:9000';
    const faveURL = 'http://127.0.0.1:9000/public/static/logo.png';

    cy.get('input[name="app.root_url"]').clear().type(rootURL);
    cy.get('input[name="app.favicon_url"]').type(faveURL);
    cy.get('.b-tabs nav a').eq(1).click();
    cy.get('.tab-item:visible').find('.field').first()
      .find('button')
      .first()
      .click();

    // Enable / disable SMTP and delete one.
    cy.get('.b-tabs nav a').eq(4).click();
    cy.get('.tab-item:visible [data-cy=btn-enable-smtp]').eq(1).click();
    cy.get('.tab-item:visible [data-cy=btn-delete-smtp]').first().click();
    cy.get('.modal button.is-primary').click();

    cy.get('[data-cy=btn-save]').click();

    cy.wait(1000);

    // Verify the changes.
    cy.request(`${apiUrl}/api/settings`).should((response) => {
      const { data } = response.body;
      expect(data['app.root_url']).to.equal(rootURL);
      expect(data['app.favicon_url']).to.equal(faveURL);
      expect(data['app.concurrency']).to.equal(9);

      expect(data.smtp.length).to.equal(1);
      expect(data.smtp[0].enabled).to.equal(true);
    });
  });
});
