const apiUrl = Cypress.env('apiUrl');
const headers = '[{"X-Custom": "Custom-Value"}]';

describe('Campaigns', () => {
  it('Opens campaigns page', () => {
    cy.resetDB();
    cy.loginAndVisit('/campaigns');
  });


  it('Counts campaigns', () => {
    cy.get('tbody td[data-label=Status]').should('have.length', 1);
  });

  it('Creates campaign', () => {
    cy.get('a[data-cy=btn-new]').click();

    // Fill fields.
    cy.get('input[name=name]').clear().type('new-attach');
    cy.get('input[name=subject]').clear().type('new-subject');
    cy.get('input[name=from_email]').clear().type('new <from@email>');
    cy.get('.list-selector input').click();
    cy.get('.list-selector .autocomplete a').eq(0).click();

    cy.get('button[data-cy=btn-continue]').click();
    cy.wait(500);

    cy.get('a[data-cy=btn-attach]').click();
    cy.get('input[type=file]').attachFile('example.json');
    cy.get('.modal button.is-primary:eq(0)').click();
    cy.wait(500);
    cy.get('.modal td[data-label=Name] a.link').click();
    cy.get('button[data-cy=btn-save]').click();
    cy.wait(500);

    // Re-open and check that the file still exists.
    cy.loginAndVisit('/campaigns');
    cy.get('td[data-label=Status] a').eq(0).click();
    cy.get('.b-tabs nav a').eq(1).click();
    cy.get('div.field[data-cy=media]').contains('example');

    // Start.
    cy.get('button[data-cy=btn-start]').click();
    cy.get('.modal button.is-primary:eq(0)').click();
    cy.wait(500);
    cy.get('tbody tr').eq(0).get('td[data-label=Status] .tag.running');
  });

  it('Edits campaign', () => {
    cy.get('td[data-label=Status] a').eq(1).click();

    // Fill fields.
    cy.get('input[name=name]').clear().type('new-name');
    cy.get('input[name=subject]').clear().type('new-subject');
    cy.get('input[name=from_email]').clear().type('new <from@email>');

    // Change the list.
    cy.get('.list-selector a.delete').click();
    cy.get('.list-selector input').click();
    cy.get('.list-selector .autocomplete a').eq(0).click();

    // Clear and redo tags.
    cy.get('input[name=tags]').type('{backspace}new-tag{enter}');

    // Enable schedule.
    cy.get('[data-cy=btn-send-later] .check').click();
    cy.wait(100);
    cy.get('.datepicker input').click();
    cy.wait(100);
    cy.get('.datepicker-header .control:nth-child(2) select').select((new Date().getFullYear() + 1).toString());
    cy.wait(100);
    cy.get('.datepicker-body a.is-selectable:first').click();
    cy.wait(100);
    cy.get('body').click(1, 1);

    // Add custom headers.
    cy.get('[data-cy=btn-headers]').click();
    cy.get('textarea[name=headers]').invoke('val', headers).trigger('input');

    // Switch to content tab.
    cy.get('.b-tabs nav a').eq(1).click();

    // Switch format to plain text.
    cy.get('label[data-cy=check-plain]').click();
    cy.get('.modal button.is-primary:eq(0)').click();

    // Enter body value.
    cy.get('textarea[name=content]').clear().type('new-content');
    cy.get('button[data-cy=btn-save]').click();

    // Schedule.
    cy.get('button[data-cy=btn-schedule]').click();
    cy.get('.modal button.is-primary:eq(0)').click();

    cy.wait(250);

    // Verify the changes.
    cy.request(`${apiUrl}/api/campaigns/1`).should((response) => {
      const { data } = response.body;
      expect(data.status).to.equal('scheduled');
      expect(data.name).to.equal('new-name');
      expect(data.subject).to.equal('new-subject');
      expect(data.content_type).to.equal('plain');
      expect(data.altbody).to.equal(null);
      expect(data.send_at).to.not.equal(null);
      expect(data.body).to.equal('new-content');

      expect(data.lists.length).to.equal(1);
      expect(data.lists[0].id).to.equal(1);
      expect(data.tags.length).to.equal(1);
      expect(data.tags[0]).to.equal('new-tag');
      expect(data.headers[0]['X-Custom']).to.equal('Custom-Value');
    });

    cy.get('tbody td[data-label=Status] .tag.scheduled');
  });


  it('Switches formats', () => {
    cy.resetDB()
    cy.loginAndVisit('/campaigns');
    const formats = ['html', 'markdown', 'plain'];
    const htmlBody = '<strong>hello</strong> \{\{ .Subscriber.Name \}\} from {\{ .Subscriber.Attribs.city \}\}';
    const plainBody = 'hello Demo Subscriber from Bengaluru';

    // Set test content the first time.
    cy.get('td[data-label=Status] a').click();
    cy.get('.b-tabs nav a').eq(1).click();
    cy.window().then((win) => {
      win.tinymce.editors[0].setContent(htmlBody);
      win.tinymce.editors[0].save();
    });
    cy.get('button[data-cy=btn-save]').click();


    formats.forEach((c) => {
      cy.loginAndVisit('/campaigns');
      cy.get('td[data-label=Status] a').click();

      // Switch to content tab.
      cy.get('.b-tabs nav a').eq(1).click();

      // Switch format.
      cy.get(`label[data-cy=check-${c}]`).click();
      cy.get('.modal button.is-primary:eq(0)').click();

      // Check content.
      cy.get('button[data-cy=btn-preview]').click();
      cy.wait(500);
      cy.get("#iframe").then(($f) => {
        if (c === 'plain') {
          return;
        }
        const doc = $f.contents();
        expect(doc.find('.wrap').text().trim().replace(/(\s|\n)+/, ' ')).equal(plainBody);
      });
      cy.get('.modal-card-foot button').click();
    });
  });


  it('Clones campaign', () => {
    cy.loginAndVisit('/campaigns');
    for (let n = 0; n < 3; n++) {
      // Clone the campaign.
      cy.get('[data-cy=btn-clone]').first().click();
      cy.get('.modal input').clear().type(`clone${n}`).click();
      cy.get('.modal button.is-primary:eq(0)').click();
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
      cy.get('.modal button.is-primary:eq(0)').click();
    });

    // Confirm deletion.
    cy.get('table tr.is-empty');
  });


  it('Adds new campaigns', () => {
    const lists = [[1], [1, 2]];
    const cTypes = ['richtext', 'html', 'markdown', 'plain'];

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

        // Add headers.
        cy.get('[data-cy=btn-headers]').click();
        cy.get('textarea[name=headers]').invoke('val', `[{"X-Header-${n}": "Value-${n}"}]`).trigger('input');

        // Hit 'Continue'.
        cy.get('button[data-cy=btn-continue]').click();
        cy.wait(250);

        // Verify the changes.
        (function (n) {
          cy.location('pathname').then((p) => {
            cy.request(`${apiUrl}/api/campaigns/${p.split('/').at(-1)}`).should((response) => {
              const { data } = response.body;
              expect(data.status).to.equal('draft');
              expect(data.name).to.equal(`name${n}`);
              expect(data.subject).to.equal(`subject${n}`);
              expect(data.content_type).to.equal('richtext');
              expect(data.altbody).to.equal(null);
              expect(data.send_at).to.equal(null);
              expect(data.headers[0][`X-Header-${n}`]).to.equal(`Value-${n}`);
            });
          });
        })(n);


        // Select content type.
        cy.get(`label[data-cy=check-${c}]`).click();

        // Insert content.
        const htmlBody = `<strong>hello${n}</strong> \{\{ .Subscriber.Name \}\} from {\{ .Subscriber.Attribs.city \}\}`;
        const plainBody = `hello${n} Demo Subscriber from Bengaluru`;
        const markdownBody = `**hello${n}** Demo Subscriber from Bengaluru`;

        cy.log(`format = ${c}`)
        if (c === 'richtext') {
          cy.window().then((win) => {
            win.tinymce.editors[0].setContent(htmlBody);
            win.tinymce.editors[0].save();
          });
          cy.wait(500);
        } else if (c === 'html') {
          cy.get('code-flask').shadow().find('.codeflask textarea').invoke('val', htmlBody).trigger('input');
        } else if (c === 'markdown') {
          cy.get('textarea[name=content]').invoke('val', markdownBody).trigger('input');
        } else if (c === 'plain') {
          cy.get('textarea[name=content]').invoke('val', plainBody).trigger('input');
        }

        // Save.
        cy.get('button[data-cy=btn-save]').click();

        // Preview and match the body.
        cy.get('button[data-cy=btn-preview]').click();
        cy.wait(1000);
        cy.get("#iframe").then(($f) => {
          if (c === 'plain') {
            return;
          }
          const doc = $f.contents();
          expect(doc.find('.wrap').text().trim()).equal(plainBody);
        });

        cy.get('.modal-card-foot button').click();

        cy.clickMenu('all-campaigns');
        cy.wait(500);

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
    cy.request(`${apiUrl}/api/campaigns?order=asc&order_by=created_at`).should((response) => {
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
      cy.get('.modal button.is-primary:eq(0)').click();
      cy.wait(250);
      cy.get(`tbody tr:nth-child(${n}) td[data-label=Status] .tag.running`);

      if (n > 1) {
        cy.get(`tbody tr:nth-child(${n}) [data-cy=btn-cancel]`).click();
        cy.get('.modal button.is-primary:eq(0)').click();
        cy.wait(250);
        cy.get(`tbody tr:nth-child(${n}) td[data-label=Status] .tag.cancelled`);
      }
    }
  });

  it('Sorts campaigns', () => {
    const asc = [5, 6, 7, 8, 9, 10, 11, 12];
    const desc = [12, 11, 10, 9, 8, 7, 6, 5];
    const cases = ['cy-name', 'cy-timestamp'];

    cases.forEach((c) => {
      cy.sortTable(`thead th.${c}`, asc);
      cy.wait(250);
      cy.sortTable(`thead th.${c}`, desc);
      cy.wait(250);
    });
  });
});
