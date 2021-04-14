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
});
