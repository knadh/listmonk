
describe('Import', () => {
  it('Opens import page', () => {
    cy.resetDB();
    cy.loginAndVisit('/subscribers/import');
  });

  it('Imports subscribers', () => {
    const cases = [
      { mode: 'check-subscribe', status: 'enabled', count: 102 },
      { mode: 'check-blocklist', status: 'blocklisted', count: 102 },
    ];

    cases.forEach((c) => {
      cy.get(`[data-cy=${c.mode}] .check`).click();

      if (c.status === 'enabled') {
        cy.get('.list-selector input').click();
        cy.get('.list-selector .autocomplete a').first().click();
      }

      cy.fixture('subs.csv').then((data) => {
        cy.get('input[type="file"]').attachFile({
          fileContent: data.toString(),
          fileName: 'subs.csv',
          mimeType: 'text/csv',
        });
      });

      cy.get('button.is-primary').click();
      cy.get('section.wrap .has-text-success');
      cy.get('button.is-primary').click();
      cy.wait(100);

      // Verify that 100 (+2 default) subs are imported.
      cy.loginAndVisit('/subscribers');
      cy.wait(100);
      cy.get('[data-cy=count]').then(($el) => {
        cy.expect(parseInt($el.text().trim())).to.equal(c.count);
      });

      cy.get('tbody td[data-label=Status]').each(($el) => {
        cy.wrap($el).find(`.tag.${c.status}`);
      });

      cy.loginAndVisit('/subscribers/import');
      cy.wait(100);
    });
  });
});
