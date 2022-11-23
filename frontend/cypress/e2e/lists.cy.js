describe('Lists', () => {
  it('Opens lists page', () => {
    cy.resetDB();
    cy.loginAndVisit('/lists');
  });


  it('Counts subscribers in default lists', () => {
    cy.get('tbody td[data-label=Subscribers]').contains('1');
  });


  it('Creates campaign for list', () => {
    cy.get('tbody a[data-cy=btn-campaign]').first().click();
    cy.location('pathname').should('contain', '/campaigns/new');
    cy.get('.list-tags .tag').contains('Default list');

    cy.clickMenu('lists', 'all-lists');
    cy.get('.modal button.is-primary').click();
  });


  it('Creates opt-in campaign for list', () => {
    cy.get('tbody a[data-cy=btn-send-optin-campaign]').click();
    cy.get('.modal button.is-primary').click();
    cy.location('pathname').should('contain', '/campaigns/2');
    cy.clickMenu('lists', 'all-lists');
  });


  it('Checks individual subscribers in lists', () => {
    const subs = [{ listID: 1, email: 'john@example.com' },
    { listID: 2, email: 'anon@example.com' }];

    // Click on each list on the lists page, go the the subscribers page
    // for that list, and check the subscriber details.
    subs.forEach((s, n) => {
      cy.get('tbody td[data-label=Subscribers] a').eq(n).click();
      cy.location('pathname').should('contain', `/subscribers/lists/${s.listID}`);
      cy.get('tbody tr').its('length').should('eq', 1);
      cy.get('tbody td[data-label="E-mail"]').contains(s.email);
      cy.clickMenu('lists', 'all-lists');
    });
  });

  it('Edits lists', () => {
    // Open the edit popup and edit the default lists.
    cy.get('[data-cy=btn-edit]').each(($el, n) => {
      cy.wrap($el).click();
      cy.get('input[name=name]').clear().type(`list-${n}`);
      cy.get('select[name=type]').select('public');
      cy.get('select[name=optin]').select('double');
      cy.get('input[name=tags]').clear().type(`tag${n}{enter}`);
      cy.get('textarea[name=description]').clear().type(`desc${n}`);
      cy.get('[data-cy=btn-save]').click();
      cy.wait(100);
    });
    cy.wait(250);

    // Confirm the edits.
    cy.get('tbody tr').each(($el, n) => {
      cy.wrap($el).find('td[data-label=Name]').contains(`list-${n}`);
      cy.wrap($el).find('.tags')
        .should('contain', 'test')
        .and('contain', `tag${n}`);
    });
  });


  it('Deletes lists', () => {
    // Delete all visible lists.
    cy.get('tbody tr').each(() => {
      cy.get('tbody a[data-cy=btn-delete]').first().click();
      cy.get('.modal button.is-primary').click();
    });

    // Confirm deletion.
    cy.get('table tr.is-empty');
  });


  // Add new lists.
  it('Adds new lists', () => {
    // Open the list form and create lists of multiple type/optin combinations.
    const types = ['private', 'public'];
    const optin = ['single', 'double'];

    let n = 0;
    types.forEach((t) => {
      optin.forEach((o) => {
        const name = `list-${t}-${o}-${n}`;

        cy.get('[data-cy=btn-new]').click();
        cy.get('input[name=name]').type(name);
        cy.get('select[name=type]').select(t);
        cy.get('select[name=optin]').select(o);
        cy.get('input[name=tags]').type(`tag${n}{enter}${t}{enter}${o}{enter}`);
        cy.get('textarea[name=description]').clear().type(`desc-${t}-${n}`);
        cy.get('[data-cy=btn-save]').click();
        cy.wait(200);

        // Confirm the addition by inspecting the newly created list row.
        const tr = `tbody tr:nth-child(${n + 1})`;
        cy.get(`${tr} td[data-label=Name]`).contains(name);
        cy.get(`${tr} td[data-label=Type] .tag[data-cy=type-${t}]`);
        cy.get(`${tr} td[data-label=Type] .tag[data-cy=optin-${o}]`);
        n++;
      });
    });
  });


  it('Searches lists', () => {
    cy.get('[data-cy=query]').clear().type('list-public-single-2{enter}');
    cy.wait(200)
    cy.get('tbody tr').its('length').should('eq', 1);
    cy.get('tbody td[data-label="Name"]').first().contains('list-public-single-2');
    cy.get('[data-cy=query]').clear().type('{enter}');
  });


  // Sort lists by clicking on various headers. At this point, there should be four
  // lists with IDs = [3, 4, 5, 6]. Sort the items be columns and match them with
  // the expected order of IDs.
  it('Sorts lists', () => {
    cy.sortTable('thead th.cy-name', [4, 3, 6, 5]);
    cy.sortTable('thead th.cy-name', [5, 6, 3, 4]);

    cy.sortTable('thead th.cy-type', [3, 4, 5, 6]);
    cy.sortTable('thead th.cy-type', [6, 5, 4, 3]);

    cy.sortTable('thead th.cy-created_at', [3, 4, 5, 6]);
    cy.sortTable('thead th.cy-created_at', [6, 5, 4, 3]);

    cy.sortTable('thead th.cy-updated_at', [3, 4, 5, 6]);
    cy.sortTable('thead th.cy-updated_at', [6, 5, 4, 3]);
  });

  it('Opens forms page', () => {
    const apiUrl = Cypress.env('apiUrl');
    cy.loginAndVisit(`${apiUrl}/subscription/form`);
    cy.get('ul li').its('length').should('eq', 2);

    const cases = [
      { 'name': 'list-public-single-2', 'description': 'desc-public-2' },
      { 'name': 'list-public-double-3', 'description': 'desc-public-3' }
    ];

    cases.forEach((c, n) => {
      cy.get('ul li').eq(n).then(($el) => {
        cy.wrap($el).get('label').contains(c.name);
        cy.wrap($el).get('.description').contains(c.description);
      });
    });
  });
});
