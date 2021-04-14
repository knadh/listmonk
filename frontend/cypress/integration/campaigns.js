describe('Subscribers', () => {
  it('Opens campaigns page', () => {
    cy.resetDB();
    cy.loginAndVisit('/campaigns');
  });


  it('Counts campaigns', () => {
    cy.get('tbody td[data-label=Status]').should('have.length', 1);
  });

  it('Edits campaign', () => {
    cy.get('td[data-label=Status] a').click();

    // Fill fields.
    cy.get('input[name=name]').clear().type('new-name');
    cy.get('input[name=subject]').clear().type('new-subject');
    cy.get('input[name=from_email]').clear().type('new <from@email>');

    // Change the list.
    cy.get('.list-selector a.delete').click();
    cy.get('.list-selector input').click();
    cy.get('.list-selector .autocomplete a').eq(1).click();

    // Clear and redo tags.
    cy.get('input[name=tags]').type('{backspace}new-tag{enter}');

    // Enable schedule.
    cy.get('[data-cy=btn-send-later] .check').click();
    cy.get('.datepicker input').click();
    cy.get('.datepicker-header .control:nth-child(2) select').select((new Date().getFullYear() + 1).toString());
    cy.get('.datepicker-body a.is-selectable:first').click();
    cy.get('body').click(1, 1);

    // Switch to content tab.
    cy.get('.b-tabs nav a').eq(1).click();

    // Switch format to plain text.
    cy.get('label[data-cy=check-plain]').click();
    cy.get('.modal button.is-primary').click();

    // Enter body value.
    cy.get('textarea[name=content]').clear().type('new-content');
    cy.get('button[data-cy=btn-save]').click();

    // Schedule.
    cy.get('button[data-cy=btn-schedule]').click();
    cy.get('.modal button.is-primary').click();

    cy.wait(250);

    // Verify the changes.
    cy.request('/api/campaigns/1').should((response) => {
      const { data } = response.body;
      expect(data.status).to.equal('scheduled');
      expect(data.name).to.equal('new-name');
      expect(data.subject).to.equal('new-subject');
      expect(data.content_type).to.equal('plain');
      expect(data.altbody).to.equal(null);
      expect(data.send_at).to.not.equal(null);
      expect(data.body).to.equal('new-content');

      expect(data.lists.length).to.equal(1);
      expect(data.lists[0].id).to.equal(2);
      expect(data.tags.length).to.equal(1);
      expect(data.tags[0]).to.equal('new-tag');
    });

    cy.get('tbody td[data-label=Status] .tag.scheduled');
  });

  it('Clones campaign', () => {
    for (let n = 0; n < 3; n++) {
      // Clone the campaign.
      cy.get('[data-cy=btn-clone]').first().click();
      cy.get('.modal input').clear().type(`clone${n}`).click();
      cy.get('.modal button.is-primary').click();
      cy.wait(250);
      cy.clickMenu('all-campaigns');
      cy.wait(100);

      // Verify the newly created row.
      cy.get('tbody td[data-label="Name"]').first().contains(`clone${n}`);
    }
  });


  it('Searches campaigns', () => {
    cy.get('input[name=query]').clear().type('clone2{enter}');
    cy.get('tbody tr').its('length').should('eq', 1);
    cy.get('tbody td[data-label="Name"]').first().contains('clone2');
    cy.get('input[name=query]').clear().type('{enter}');
  });


  it('Deletes campaign', () => {
    // Delete all visible lists.
    cy.get('tbody tr').each(() => {
      cy.get('tbody a[data-cy=btn-delete]').first().click();
      cy.get('.modal button.is-primary').click();
    });

    // Confirm deletion.
    cy.get('table tr.is-empty');
  });


  it('Adds new campaigns', () => {
    const lists = [[1], [1, 2]];
    const cTypes = ['richtext', 'html', 'plain'];

    let n = 0;
    cTypes.forEach((c) => {
      lists.forEach((l) => {
      // Click the 'new button'
        cy.get('[data-cy=btn-new]').click();
        cy.wait(100);

        // Fill fields.
        cy.get('input[name=name]').clear().type(`name${n}`);
        cy.get('input[name=subject]').clear().type(`subject${n}`);

        l.forEach(() => {
          cy.get('.list-selector input').click();
          cy.get('.list-selector .autocomplete a').first().click();
        });

        // Add tags.
        for (let i = 0; i < 3; i++) {
          cy.get('input[name=tags]').type(`tag${i}{enter}`);
        }

        // Hit 'Continue'.
        cy.get('button[data-cy=btn-continue]').click();
        cy.wait(250);

        // Insert content.
        cy.get('.ql-editor').type(`hello${n} \{\{ .Subscriber.Name \}\}`, { parseSpecialCharSequences: false });
        cy.get('.ql-editor').type('{enter}');
        cy.get('.ql-editor').type('\{\{ .Subscriber.Attribs.city \}\}', { parseSpecialCharSequences: false });

        // Select content type.
        cy.get(`label[data-cy=check-${c}]`).click();

        // If it's not richtext, there's a "you'll lose formatting" prompt.
        if (c !== 'richtext') {
          cy.get('.modal button.is-primary').click();
        }

        // Save.
        cy.get('button[data-cy=btn-save]').click();

        cy.clickMenu('all-campaigns');
        cy.wait(250);

        // Verify the newly created campaign in the table.
        cy.get('tbody td[data-label="Name"]').first().contains(`name${n}`);
        cy.get('tbody td[data-label="Name"]').first().contains(`subject${n}`);
        cy.get('tbody td[data-label="Lists"]').first().then(($el) => {
          cy.wrap($el).find('li').should('have.length', l.length);
        });

        n++;
      });
    });

    // Fetch the campaigns API and verfiy the values that couldn't be verified on the table UI.
    cy.request('/api/campaigns?order=asc&order_by=created_at').should((response) => {
      const { data } = response.body;
      expect(data.total).to.equal(lists.length * cTypes.length);

      let n = 0;
      cTypes.forEach((c) => {
        lists.forEach((l) => {
          expect(data.results[n].content_type).to.equal(c);
          expect(data.results[n].lists.map((ls) => ls.id)).to.deep.equal(l);
          n++;
        });
      });
    });
  });

  it('Starts and cancels campaigns', () => {
    for (let n = 1; n <= 2; n++) {
      cy.get(`tbody tr:nth-child(${n}) [data-cy=btn-start]`).click();
      cy.get('.modal button.is-primary').click();
      cy.wait(250);
      cy.get(`tbody tr:nth-child(${n}) td[data-label=Status] .tag.running`);

      if (n > 1) {
        cy.get(`tbody tr:nth-child(${n}) [data-cy=btn-cancel]`).click();
        cy.get('.modal button.is-primary').click();
        cy.wait(250);
        cy.get(`tbody tr:nth-child(${n}) td[data-label=Status] .tag.cancelled`);
      }
    }
  });

  it('Sorts campaigns', () => {
    const asc = [5, 6, 7, 8, 9, 10];
    const desc = [10, 9, 8, 7, 6, 5];
    const cases = ['cy-name', 'cy-timestamp'];

    cases.forEach((c) => {
      cy.sortTable(`thead th.${c}`, asc);
      cy.wait(250);
      cy.sortTable(`thead th.${c}`, desc);
      cy.wait(250);
    });
  });
});
