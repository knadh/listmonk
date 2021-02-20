describe('Dashboard', () => {
  it('Opens dashboard', () => {
    cy.loginAndVisit('/');

    // List counts.
    cy.get('[data-cy=lists]')
      .should('contain', '2 Lists')
      .and('contain', '1 Public')
      .and('contain', '1 Private')
      .and('contain', '1 Single opt-in')
      .and('contain', '1 Double opt-in');

    // Campaign counts.
    cy.get('[data-cy=campaigns]')
      .should('contain', '1 Campaign')
      .and('contain', '1 draft');

    // Subscriber counts.
    cy.get('[data-cy=subscribers]')
      .should('contain', '2 Subscribers')
      .and('contain', '0 Blocklisted')
      .and('contain', '0 Orphans');

    // Message count.
    cy.get('[data-cy=messages]')
      .should('contain', '0 Messages sent');
  });
});
