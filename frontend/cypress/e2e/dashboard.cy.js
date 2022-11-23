describe('Dashboard', () => {
  it('Opens dashboard', () => {
    cy.resetDB();
    cy.loginAndVisit('/');

    // List counts.
    cy.get('[data-cy=lists] .title').contains('2');
    cy.get('[data-cy=lists]')
      .and('contain', '1 Public')
      .and('contain', '1 Private')
      .and('contain', '1 Single opt-in')
      .and('contain', '1 Double opt-in');

    // Campaign counts.
    cy.get('[data-cy=campaigns] .title').contains('1');
    cy.get('[data-cy=campaigns-draft]').contains('1');

    // Subscriber counts.
    cy.get('[data-cy=subscribers] .title').contains('2');
    cy.get('[data-cy=subscribers]')
      .should('contain', '0 Blocklisted')
      .and('contain', '0 Orphans');

    // Message count.
    cy.get('[data-cy=messages] .title').contains('0');
  });
});
