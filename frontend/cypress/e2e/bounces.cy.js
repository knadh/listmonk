const apiUrl = Cypress.env('apiUrl');

describe('Bounces', () => {
  let subs = [];

  it('Enable bounces', () => {
    cy.resetDB();

    cy.loginAndVisit('/settings');
    cy.get('.b-tabs nav a').eq(5).click();
    cy.get('[data-cy=btn-enable-bounce] .switch').click();
    cy.get('[data-cy=btn-enable-bounce-webhook] .switch').click();
    cy.get('[data-cy=btn-bounce-count] .plus').click();

    cy.get('[data-cy=btn-save]').click();
    cy.wait(2000);
  });


  it('Post bounces', () => {
    // Get campaign.
    let camp = {};
    cy.request(`${apiUrl}/api/campaigns`).then((resp) => {
      camp = resp.body.data.results[0];
    })
    cy.then(() => {
      console.log("campaign is ", camp.uuid);
    })


    // Get subscribers.
    cy.request(`${apiUrl}/api/subscribers`).then((resp) => {
      subs = resp.body.data.results;
      console.log(subs)
    });

    cy.then(() => {
      console.log(`got ${subs.length} subscribers`);

      // Post bounces. Blocklist the 1st sub.
      cy.request('POST', `${apiUrl}/webhooks/bounce`, { source: "api", type: "hard", email: subs[0].email });
      cy.request('POST', `${apiUrl}/webhooks/bounce`, { source: "api", type: "hard", campaign_uuid: camp.uuid, email: subs[0].email });
      cy.request('POST', `${apiUrl}/webhooks/bounce`, { source: "api", type: "hard", campaign_uuid: camp.uuid, subscriber_uuid: subs[0].uuid });

      for (let i = 0; i < 2; i++) {
        cy.request('POST', `${apiUrl}/webhooks/bounce`, { source: "api", type: "soft", campaign_uuid: camp.uuid, subscriber_uuid: subs[1].uuid });
      }
    });

    cy.wait(250);
  });

  it('Opens bounces page', () => {
    cy.loginAndVisit('/subscribers/bounces');
    cy.wait(250);
    cy.get('tbody tr').its('length').should('eq', 5);
  });

  it('Delete bounce', () => {
    cy.get('tbody tr:last-child [data-cy="btn-delete"]').click();
    cy.get('.modal button.is-primary').click();
    cy.wait(250);
    cy.get('tbody tr').its('length').should('eq', 4);
  });

  it('Check subscriber statuses', () => {
    cy.loginAndVisit(`/subscribers/${subs[0].id}`);
    cy.wait(250);
    cy.get('.modal-card-head .tag').should('have.class', 'blocklisted');
    cy.get('.modal-card-foot button[type="button"]').click();

    cy.loginAndVisit(`/subscribers/${subs[1].id}`);
    cy.wait(250);
    cy.get('.modal-card-head .tag').should('have.class', 'enabled');
  });

});
