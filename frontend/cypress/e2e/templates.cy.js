describe('Templates', () => {
  it('Opens templates page', () => {
    cy.resetDB();
    cy.loginAndVisit('/campaigns/templates');
  });


  it('Counts default templates', () => {
    cy.get('tbody td[data-label=Name]').should('have.length', 3);
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
    cy.get('code-flask').shadow().find('.codeflask textarea').invoke('val', '<span>test</span> {{ template "content" . }}').trigger('input');

    cy.get('.modal-card-foot button.is-primary').click();
    cy.wait(250);
    cy.get('tbody td[data-label="Name"] a').contains('edited');
  });


  it('Previews campaign templates', () => {
    // Edited one sould have a bare body.
    cy.get('tbody [data-cy=btn-preview').eq(0).click();
    cy.wait(500);
    cy.get('.modal-card-body iframe').iframe(() => {
      cy.get('span').first().contains('test');
      cy.get('p').first().contains('Hi there');
    });
    cy.get('.modal-card-foot button').click();

    // Cloned one should have the full template.
    cy.get('tbody [data-cy=btn-preview').eq(3).click();
    cy.wait(500);
    cy.get('.modal-card-body iframe').iframe(() => {
      cy.get('.wrap p').first().contains('Hi there');
      cy.get('.footer a').first().contains('Unsubscribe');
    });
    cy.get('.modal-card-foot button').click();
  });

  it('Previews tx templates', () => {
    cy.get('tbody td[data-label="Name"]').contains('td', 'cloned tx').then((el) => {
      cy.wrap(el).parent().find('[data-cy=btn-preview]').click();
      cy.wait(500);
      cy.get('.modal-card-body iframe').iframe(() => {
        cy.get('strong').first().contains('Order number');
      });
      cy.get('.modal-card-foot button').click();
    });
  });

  it('Sets default', () => {
    cy.get('tbody td[data-label="Name"]').contains('td', 'cloned campaign').then((el) => {
      cy.wrap(el).parent().find('[data-cy=btn-set-default]').click();
      cy.get('.modal button.is-primary').click();

    });

    // The original default shouldn't have default and the new one should have.
    cy.get('tbody').contains('td', 'edited').parent().find('[data-cy=btn-delete]').should('exist');
    cy.get('tbody').contains('td', 'cloned campaign').parent().find('[data-cy=btn-delete]').should('not.exist');
  });


  it('Deletes template', () => {
    cy.wait(250);

    ['Default archive template', 'Sample transactional template'].forEach((t) => {
      cy.get('tbody td[data-label="Name"]').contains('td', t).then((el) => {
        cy.wrap(el).parent().find('[data-cy=btn-delete]').click();
        cy.get('.modal button.is-primary').click();
      });
      cy.wait(250);
    })

    cy.get('tbody td.actions').should('have.length', 3);
  });
});
