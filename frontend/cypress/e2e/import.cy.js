describe('Import', () => {
  it('Opens import page', () => {
    cy.resetDB();
    cy.loginAndVisit('/admin/subscribers/import');
  });

  it('Imports subscribers', () => {
    const cases = [
      {
        chkMode: 'subscribe', status: 'enabled', chkSubStatus: 'unconfirmed', subStatus: 'unconfirmed', overwrite: true, count: 102,
      },
      {
        chkMode: 'subscribe', status: 'enabled', chkSubStatus: 'confirmed', subStatus: 'confirmed', overwrite: true, count: 102,
      },
      {
        chkMode: 'subscribe', status: 'enabled', chkSubStatus: 'unconfirmed', subStatus: 'confirmed', overwrite: false, count: 102,
      },
      {
        chkMode: 'blocklist', status: 'blocklisted', chkSubStatus: 'unsubscribed', subStatus: 'unsubscribed', overwrite: false, count: 102,
      },
    ];

    cases.forEach((c) => {
      cy.get(`[data-cy=check-${c.chkMode}] .check`).click();
      cy.get(`[data-cy=check-${c.chkSubStatus}] .check`).click();

      if (c.overwrite) {
        cy.get('[data-cy=overwrite-user-info]').click();
        cy.get('[data-cy=overwrite-sub-status]').click();
      }

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

      // ONLY if .modal button.is-primary is present, click it.
      if (c.overwrite) {
        cy.get('.modal button.is-primary').click();
      }

      cy.get('section.wrap .has-text-success');
      cy.get('button.is-primary').click();
      cy.wait(100);

      // Verify that 100 (+2 default) subs are imported.
      cy.loginAndVisit('/admin/subscribers');
      cy.wait(100);
      cy.get('[data-cy=count]').then(($el) => {
        cy.expect(parseInt($el.text().trim())).to.equal(c.count);
      });

      // Subscription status.
      // cy.get('tbody td[data-label=E-mail]').each(($el) => {
      // cy.wrap($el).find(`.tag.${c.subStatus}`);
      // });

      cy.loginAndVisit('/admin/subscribers/import');
      cy.wait(100);
    });
  });

  it('Imports subscribers incorrectly', () => {
    cy.wait(1000);
    cy.resetDB();
    cy.wait(1000);
    cy.loginAndVisit('/admin/subscribers/import');

    cy.get('.list-selector input').click();
    cy.get('.list-selector .autocomplete a').first().click();
    cy.get('input[name=delim]').clear().type('|');

    cy.fixture('subs.csv').then((data) => {
      cy.get('input[type="file"]').attachFile({
        fileContent: data.toString(),
        fileName: 'subs.csv',
        mimeType: 'text/csv',
      });
    });

    cy.get('button.is-primary').click();
    cy.wait(250);
    cy.get('section.wrap .has-text-danger');
  });
});
