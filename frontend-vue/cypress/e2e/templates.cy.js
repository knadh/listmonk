describe('Templates', () => {
  it('Opens templates page', () => {
    cy.resetDB();
    cy.loginAndVisit('/admin/campaigns/templates');
  });

  it('Counts default templates', () => {
    cy.get('tbody td[data-label=Name]').should('have.length', 4);
  });

  it('Clones campaign template', () => {
    cy.get('[data-cy=btn-clone]').first().click();
    cy.get('.modal input').clear().type('cloned campaign').click();
    cy.get('.modal button.is-primary').click();
    cy.wait(250);

    // Verify the newly created row.
    cy.get('tbody td[data-label="Name"]').contains('td', 'cloned campaign');
  });

  it('Clones tx template', () => {
    cy.get('tbody td[data-label="Name"]').contains('td', 'Sample transactional template').then((el) => {
      cy.wrap(el).parent().find('[data-cy=btn-clone]').click();
      cy.get('.modal input').clear().type('cloned tx').click();
      cy.get('.modal button.is-primary').click();
      cy.wait(250);
    });

    // Verify the newly created row.
    cy.get('tbody td[data-label="Name"]').contains('td', 'cloned tx');
  });

  it('Edits template', () => {
    cy.get('tbody td.actions [data-cy=btn-edit]').first().click();
    cy.wait(250);
    cy.get('input[name=name]').clear().type('edited');

    const htmlBody = '<span>test</span><div class="wrap">{{ template "content" . }}</div>';
    cy.get('[role="textbox"]').invoke('text', htmlBody);

    cy.get('.modal-card-foot button.is-primary').click();
    cy.wait(250);
    cy.get('tbody td[data-label="Name"] a').contains('edited');
  });

  it('Previews campaign templates', () => {
    const apiUrl = Cypress.env('apiUrl');

    // Edited one should have a bare body.
    cy.request(`${apiUrl}/api/templates/1/preview`).then((resp) => {
      expect(resp.body).to.contain('test');
      expect(resp.body).to.contain('Hi there');
    });

    // Cloned one should have the full template with wrap and unsubscribe.
    cy.request(`${apiUrl}/api/templates/5/preview`).then((resp) => {
      expect(resp.body).to.contain('Hi there');
      expect(resp.body).to.contain('Unsubscribe');
    });
  });

  it('Previews tx templates', () => {
    const apiUrl = Cypress.env('apiUrl');

    // Cloned tx template.
    cy.request(`${apiUrl}/api/templates/6/preview`).then((resp) => {
      expect(resp.body).to.contain('Order number');
    });
  });

  it('Sets default', () => {
    cy.get('tbody td[data-label="Name"]').contains('td', 'cloned campaign').then((el) => {
      cy.wrap(el).parent().find('[data-cy=btn-set-default]').click();
      cy.get('.modal button.is-primary').click();
    });

    // The original default shouldn't have default and the new one should have.
    cy.get('tbody').contains('td', 'edited').parent().find('[data-cy=btn-delete]')
      .should('exist');
    cy.get('tbody').contains('td', 'cloned campaign').parent().find('[data-cy=btn-delete]')
      .should('not.exist');
  });

  it('Deletes template', () => {
    cy.wait(250);

    ['Default archive template', 'Sample transactional template'].forEach((t) => {
      cy.get('tbody td[data-label="Name"]').contains('td', t).then((el) => {
        cy.wrap(el).parent().find('[data-cy=btn-delete]').click();
        cy.get('.modal button.is-primary').click();
      });
      cy.wait(250);
    });

    cy.get('tbody td.actions').should('have.length', 4);
  });
});
