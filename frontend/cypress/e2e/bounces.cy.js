const apiUrl = Cypress.env('apiUrl');

describe('Bounces', () => {
  let subs = [];

  it('Enable bounces', () => {
    cy.resetDB();

    cy.loginAndVisit('/settings');
    cy.get('.b-tabs nav a').eq(6).click();
    cy.get('[data-cy=btn-enable-bounce] .switch').click();
    cy.get('[data-cy=btn-enable-bounce-webhook] .switch').click();

    cy.get('[data-cy=btn-save]').click();
    cy.wait(2000);
  });

  it('Post bounces', () => {
    // Get campaign.
    let camp = {};
    cy.request(`${apiUrl}/api/campaigns`).then((resp) => {
      camp = resp.body.data.results[0];
    }).then(() => {
      console.log('campaign is ', camp.uuid);
    });

    // Get subscribers.
    let subs = [];
    cy.request(`${apiUrl}/api/subscribers`).then((resp) => {
      subs = resp.body.data.results;
    }).then(() => {
      // Register soft bounces do nothing.
      let sub = {};
      cy.request('POST', `${apiUrl}/webhooks/bounce`, { source: 'api', type: 'soft', email: subs[0].email });
      cy.request('POST', `${apiUrl}/webhooks/bounce`, { source: 'api', type: 'soft', email: subs[0].email });
      cy.request(`${apiUrl}/api/subscribers/${subs[0].id}`).then((resp) => {
        sub = resp.body.data;
      }).then(() => {
        cy.expect(sub.status).to.equal('enabled');
      });

      // Hard bounces blocklist.
      cy.request('POST', `${apiUrl}/webhooks/bounce`, { source: 'api', type: 'hard', email: subs[0].email });
      cy.request('POST', `${apiUrl}/webhooks/bounce`, { source: 'api', type: 'hard', email: subs[0].email });
      cy.request(`${apiUrl}/api/subscribers/${subs[0].id}`).then((resp) => {
        sub = resp.body.data;
      }).then(() => {
        cy.expect(sub.status).to.equal('blocklisted');
      });

      // Complaint bounces delete.
      cy.request('POST', `${apiUrl}/webhooks/bounce`, { source: 'api', type: 'complaint', email: subs[1].email });
      cy.request('POST', `${apiUrl}/webhooks/bounce`, { source: 'api', type: 'complaint', email: subs[1].email });
      cy.request({ url: `${apiUrl}/api/subscribers/${subs[1].id}`, failOnStatusCode: false }).then((resp) => {
        expect(resp.status).to.eq(400);
      });

      cy.loginAndVisit('/subscribers/bounces');
    });
  });
});
