const apiUrl = Cypress.env('apiUrl');

describe('Forms', () => {
  it('Opens forms page', () => {
    cy.resetDB();
    cy.loginAndVisit('/lists/forms');
  });

  it('Checks form URL', () => {
    cy.get('a[data-cy=url]').contains('http://localhost:9000');
  });

  it('Checks public lists', () => {
    cy.get('ul[data-cy=lists] li')
      .should('contain', 'Opt-in list')
      .its('length')
      .should('eq', 1);

    cy.get('[data-cy=form] pre').should('not.exist');
  });

  it('Selects public list', () => {
    // Click the list checkbox.
    cy.get('ul[data-cy=lists] .checkbox').click();

    // Make sure the <pre> form HTML has appeared.
    cy.get('[data-cy=form] pre').then(($pre) => {
      // Check that the ID of the list in the checkbox appears in the HTML.
      cy.get('ul[data-cy=lists] input').then(($inp) => {
        cy.wrap($pre).contains($inp.val());
      });
    });

    // Click the list checkbox.
    cy.get('ul[data-cy=lists] .checkbox').click();
    cy.get('[data-cy=form] pre').should('not.exist');
  });

  it('Subscribes from public form page', () => {
    // Create a public test list.
    cy.request('POST', `${apiUrl}/api/lists`, { name: 'test-list', type: 'public', optin: 'single' });

    // Open the public page and subscribe to alternating lists multiple times.
    // There should be no errors and two new subscribers should be subscribed to two lists.
    for (let i = 0; i < 2; i++) {
      for (let j = 0; j < 2; j++) {
        cy.loginAndVisit(`${apiUrl}/subscription/form`);
        cy.get('input[name=email]').clear().type(`test${i}@test.com`);
        cy.get('input[name=name]').clear().type(`test${i}`);
        cy.get('input[type=checkbox]').eq(j).click();
        cy.get('button').click();
        cy.wait(250);
        cy.get('.wrap').contains(/has been sent|successfully/);
      }
    }

    // Verify form subscriptions.
    cy.request(`${apiUrl}/api/subscribers`).should((response) => {
      const { data } = response.body;

      // Two new + two dummy subscribers that are there by default.
      expect(data.total).to.equal(4);

      // The two new subscribers should each have two list subscriptions.
      for (let i = 0; i < 2; i++) {
        expect(data.results.find((s) => s.email === `test${i}@test.com`).lists.length).to.equal(2);
      }
    });
  });
});
