describe('Campaign delivery metrics', () => {
  before(() => {
    cy.resetDB();
    cy.loginAndVisit('/admin/campaigns');
  });

  it('Shows formulas, zero denominators, and estimated delivery', () => {
    let scenario = 'rates';
    cy.intercept('GET', '**/api/campaigns/running/stats', { body: { data: [] } });
    cy.intercept({ method: 'GET', pathname: '/api/campaigns' }, (req) => {
      req.continue((res) => {
        const campaign = res.body.data.results[0];
        if (scenario === 'zero') {
          Object.assign(campaign, {
            to_send: 0,
            sent: 0,
            delivered: 0,
            unique_views: 0,
            unique_clicks: 0,
            bounces: 0,
            delivery_estimated: false,
          });
          return;
        }

        Object.assign(campaign, {
          to_send: 200,
          sent: 100,
          delivered: scenario === 'estimated' ? 90 : 80,
          unique_views: 40,
          unique_clicks: 20,
          bounces: 5,
          delivery_estimated: scenario === 'estimated',
        });
      });
    });
    cy.intercept('GET', '**/api/config', (req) => {
      req.continue((res) => {
        res.body.data.privacy.individual_tracking = true;
      });
    });

    cy.visit('/admin/campaigns');
    cy.get('tbody td[data-label=Stats]').first().within(() => {
      cy.contains('p', 'Views').should('contain', '40').and('contain', '50.0%');
      cy.contains('p', 'Clicks').should('contain', '20').and('contain', '25.0%');
      cy.contains('p', 'Delivered').should('contain', '80').and('contain', '80.0%');
      cy.contains('p', 'Sent').should('contain', '100').and('contain', '200').and('contain', '50.0%');
      cy.contains('p', 'Bounces').should('contain', '5').and('contain', '5.0%');
    });

    cy.then(() => {
      scenario = 'zero';
    });
    cy.reload();
    cy.get('tbody td[data-label=Stats]').first()
      .find('.stat-percentage').should('have.length', 5)
      .each(($percentage) => cy.wrap($percentage).should('contain', '—'));

    cy.then(() => {
      scenario = 'estimated';
    });
    cy.reload();
    cy.get('tbody td[data-label=Stats]').first().within(() => {
      cy.contains('p', 'Delivered').should('contain', '~90');
      cy.contains('Estimated').should('exist');
    });
  });

  it('Defaults to Unique, changes in Total mode, and ignores stale responses', () => {
    cy.intercept('GET', '**/api/config', (req) => {
      req.continue((res) => {
        res.body.data.privacy.individual_tracking = true;
      });
    });

    cy.intercept('GET', '**/api/campaigns/analytics/views*', (req) => {
      const isTotal = req.query.mode === 'total';
      if (isTotal) {
        req.alias = 'delayedTotalViews';
      }
      req.reply({
        delay: isTotal ? 500 : 10,
        body: {
          data: [{ campaign_id: 1, timestamp: '2026-08-18T00:00:00Z', count: isTotal ? 99 : 7 }],
        },
      });
    });

    cy.visit('/admin/campaigns/analytics?id=1');
    cy.get('[data-cy=analytics-mode-unique]').should('have.class', 'is-primary').and('not.be.disabled');
    cy.get('[data-cy=analytics-mode-total]').should('have.class', 'is-light');
    cy.contains('.chart h4', 'Unique Views').should('contain', '(7)');

    cy.get('[data-cy=analytics-mode-total]').click();
    cy.wait('@delayedTotalViews');
    cy.contains('.chart h4', 'Total Views').should('contain', '(99)');
    cy.get('[data-cy=analytics-mode-unique]').click();
    cy.contains('.chart h4', 'Unique Views').should('contain', '(7)');

    // The older Total response deliberately finishes after the newer Unique
    // response. It must not overwrite the current mode's chart or count.
    cy.get('[data-cy=analytics-mode-total]').click();
    cy.get('[data-cy=analytics-mode-unique]').click();
    cy.wait('@delayedTotalViews');
    cy.contains('.chart h4', 'Unique Views').should('contain', '(7)');
  });

  it('Refreshes campaign metrics while visible and resumes after tab visibility changes', () => {
    // Leave the Campaigns page before installing the clock so its existing
    // timers are cleaned up with the native timer functions.
    cy.get('.menu a').first().click();
    cy.clock();
    cy.intercept('GET', '**/api/campaigns/running/stats', { body: { data: [] } });
    cy.intercept({ method: 'GET', pathname: '/api/campaigns' }).as('campaignList');
    cy.get('.menu a[data-cy="all-campaigns"]').click({ force: true });

    cy.wait('@campaignList');
    cy.tick(30000);
    cy.wait('@campaignList');

    let hidden = false;
    cy.document().then((doc) => {
      Object.defineProperty(doc, 'hidden', { configurable: true, get: () => hidden });
      hidden = true;
      doc.dispatchEvent(new Event('visibilitychange'));
    });
    cy.tick(60000);
    cy.get('@campaignList.all').should('have.length', 2);

    cy.document().then((doc) => {
      hidden = false;
      doc.dispatchEvent(new Event('visibilitychange'));
    });
    cy.wait('@campaignList');
    cy.tick(30000);
    cy.wait('@campaignList');

    // Destroying Campaigns.vue must stop future listing refreshes.
    cy.get('.menu a').first().click();
    cy.tick(60000);
    cy.get('@campaignList.all').should('have.length', 4);
  });
});
