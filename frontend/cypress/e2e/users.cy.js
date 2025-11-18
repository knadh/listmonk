const apiUrl = Cypress.env('apiUrl');

describe('First time user setup', () => {
  it('Sets up the superadmin user', () => {
    cy.resetDBBlank();
    cy.visit('/admin/login');

    cy.get('input[name=email]').type('super@domain');
    cy.get('input[name=username]').type('super');
    cy.get('input[name=password]').type('super123');
    cy.get('input[name=password2]').type('super123');
    cy.get('button[type=submit]').click();
    cy.wait(500);
    cy.visit('/admin/users');

    cy.get('[data-cy=btn-edit]').first().click();
    cy.get('select[name=user_role]').should('have.value', '1');
  });
});

describe('User roles', () => {
  it('Opens user roles page', () => {
    cy.resetDB();
    cy.loginAndVisit('/admin/users/roles/users');
  });

  it('Adds new roles', () => {
    // first - no global list perms.
    cy.get('[data-cy=btn-new]').click();
    cy.get('input[name=name]').type('first');
    cy.get('[data-cy=btn-save]').click();
    cy.wait(500);

    // second - all perms.
    cy.get('[data-cy=btn-new]').click();
    cy.get('input[name=name]').type('second');
    cy.get('input[type=checkbox]').each((e) => {
      cy.get(e).check({ force: true });
    });
    cy.get('[data-cy=btn-save]').click();
    cy.wait(200);
  });

  it('Edits role', () => {
    cy.get('[data-cy=btn-edit]').first().click();
    cy.get('input[value="users:get"]').check({ force: true });
    cy.get('[data-cy=btn-save]').click();
  });

  it('Deletes role', () => {
    cy.get('[data-cy=btn-clone]').last().click();
    cy.get('.modal-card-foot button.is-primary').click();
    cy.wait(500);
    cy.get('[data-cy=btn-delete]').last().click();
    cy.get('.modal button.is-primary').click();

    cy.get('tbody tr').should('have.length', 3);
  });
});

describe('List roles', () => {
  it('Opens roles page', () => {
    cy.loginAndVisit('/admin/users/roles/lists');
  });

  it('Adds new roles', () => {
    cy.get('[data-cy=btn-new]').click();
    cy.get('input[name=name]').type('first');
    cy.get('.box button.is-primary').click();
    cy.get('[data-cy=btn-save]').click();
    cy.wait(500);

    cy.get('[data-cy=btn-new]').click();
    cy.get('input[name=name]').type('second');
    cy.get('.box button.is-primary').click();
    cy.get('.box button.is-primary').click();
    cy.get('[data-cy=btn-save]').click();
    cy.wait(500);
  });

  it('Edits role', () => {
    cy.get('[data-cy=btn-edit]').eq(1).click();

    // Uncheck "manage" permission on the second item.
    cy.get('input[type=checkbox]').eq(3).uncheck({ force: true });
    cy.get('[data-cy=btn-save]').click();
  });

  it('Deletes role', () => {
    cy.get('[data-cy=btn-clone]').last().click();
    cy.get('.modal-card-foot button.is-primary').click();
    cy.wait(500);
    cy.get('[data-cy=btn-delete]').last().click();
    cy.get('.modal button.is-primary').click();

    cy.get('tbody tr').should('have.length', 2);
  });
});

describe('Users ', () => {
  it('Opens users page', () => {
    cy.loginAndVisit('/admin/users');
  });

  it('Adds new users', () => {
    ['first', 'second', 'third'].forEach((name) => {
      cy.get('[data-cy=btn-new]').click();
      cy.get('input[name=username]').type(name);
      cy.get('input[name=name]').type(name);
      cy.get('input[name=email]').type(`${name}@domain`);
      cy.get('input[name=password_login]').check({ force: true });
      cy.get('input[name=password]').type(`${name}000000`);
      cy.get('input[name=password2]').type(`${name}000000`);

      const role = name !== 'third' ? name : 'first';
      cy.get('select[name=user_role]').select(role);
      cy.get('select[name=list_role]').select(role);
      cy.get('.modal button.is-primary').click();
      cy.wait(500);
    });
  });

  it('Edits user', () => {
    cy.get('[data-cy=btn-edit]').last().click();
    cy.get('input[name=password_login]').uncheck({ force: true });
    cy.get('select[name=user_role]').select('second');
    cy.get('select[name=list_role]').select('second');
    cy.get('.modal button.is-primary').click();
    cy.wait(500);

    // Fetch the campaigns API and verfiy the values that couldn't be verified on the table UI.
    cy.request(`${apiUrl}/api/users/4`).should((response) => {
      const { data } = response.body;

      expect(data.password_login).to.equal(false);
      expect(data.user_role.name).to.equal('second');
      expect(data.list_role.name).to.equal('second');
    });
  });

  it('Deletes a user', () => {
    cy.get('[data-cy=btn-delete]').last().click();
    cy.get('.modal-card-foot button.is-primary').click();
    cy.wait(500);
    cy.get('tbody tr').should('have.length', 3);
  });
});

describe('Login ', () => {
  it('Logs in as first', () => {
    cy.visit('/admin/login?next=/admin/lists');
    cy.get('input[name=username]').invoke('val', 'first');
    cy.get('input[name=password]').invoke('val', 'first000000');
    cy.get('button').click();

    // first=only default list.
    cy.get('tbody tr').should('have.length', 1);
    cy.get('tbody td[data-label=Name]').contains('Default list');
    cy.get('[data-cy=btn-new]').should('not.exist');
    cy.get('[data-cy=btn-edit]').should('exist');
    cy.get('[data-cy=btn-delete]').should('exist');
  });

  it('Logs in as second', () => {
    cy.visit('/admin/login?next=/admin/lists');
    cy.get('input[name=username]').invoke('val', 'second');
    cy.get('input[name=password]').invoke('val', 'second000000');
    cy.get('button').click();

    // first=only default list.
    cy.get('tbody tr').should('have.length', 2);
    cy.get('tbody tr:nth-child(1) [data-cy=btn-edit]').should('exist');
    cy.get('tbody tr:nth-child(1) [data-cy=btn-delete]').should('exist');
    cy.get('tbody tr:nth-child(2) [data-cy=btn-edit]').should('exist');
    cy.get('tbody tr:nth-child(2) [data-cy=btn-delete]').should('exist');
  });
});
