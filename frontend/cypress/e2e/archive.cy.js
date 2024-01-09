const apiUrl = Cypress.env('apiUrl');

describe('Archive', () => {
  it('Opens campaigns page', () => {
    cy.resetDB();
    cy.loginAndVisit('/campaigns');
    cy.wait(500);
  });

  it('Clones campaign', () => {
    cy.loginAndVisit('/campaigns');
    cy.get('[data-cy=btn-clone]').first().click();
    cy.get('.modal input').clear().type('clone').click();
    cy.get('.modal button.is-primary').click();
    cy.wait(250);

    cy.loginAndVisit('/campaigns');
    cy.get('[data-cy=btn-clone]').first().click();
    cy.get('.modal input').clear().type('clone2').click();
    cy.get('.modal button.is-primary').click();
    cy.wait(250);

    cy.clickMenu('all-campaigns');
  });

  it('Starts campaigns', () => {
    cy.get('td[data-label=Status] a').eq(0).click();
    cy.get('[data-cy=btn-start]').click();
    cy.get('.modal button.is-primary').click();

    cy.get('td[data-label=Status] a').eq(1).click();
    cy.get('[data-cy=btn-start]').click();
    cy.get('.modal button.is-primary').click();
    cy.wait(1000);
  });

  it('Enables archive on one campaign (no slug)', () => {
    cy.loginAndVisit('/campaigns');
    cy.wait(250);
    cy.get('td[data-label=Status] a').eq(0).click();

    // Switch to archive tab and enable archive.
    cy.get('.b-tabs nav a').eq(2).click();
    cy.wait(500);
    cy.get('[data-cy=btn-archive] .check').click();
    cy.get('[data-cy=archive-slug]').clear();
    cy.get('[data-cy=archive-meta]').clear()
      .type('{"email": "archive@domain.com", "name": "Archive", "attribs": { "city": "Bengaluru"}}', { parseSpecialCharSequences: false });
    cy.get('[data-cy=btn-save]').click();
    cy.wait(250);
  });

  it('Enables archive on one campaign', () => {
    cy.loginAndVisit('/campaigns');
    cy.wait(250);
    cy.get('td[data-label=Status] a').eq(1).click();

    // Switch to archive tab and enable archive.
    cy.get('.b-tabs nav a').eq(2).click();
    cy.wait(500);
    cy.get('[data-cy=btn-archive] .check').click();
    cy.get('[data-cy=archive-slug]').clear().type('my-archived-campaign');
    cy.get('[data-cy=btn-save]').click();
    cy.wait(250);
  });

  it('Opens campaign archive page', () => {
    for (let i = 0; i < 2; i++) {
      cy.loginAndVisit(`${apiUrl}/archive`);
      cy.get('li a').eq(i).click();
      cy.wait(250);
      if (i === 0) {
        cy.get('h3').contains('Hi Archive!');
        cy.get('p').eq(0).contains('Bengaluru');
      } else {
        cy.get('h3').contains('Hi Subscriber!');
      }
    }
  });
});
