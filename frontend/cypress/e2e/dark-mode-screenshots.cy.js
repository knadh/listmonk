describe('Dark Mode Screenshots', () => {
  it('Takes screenshots of login page and dashboard in light and dark modes', () => {
    // Take screenshot of login page (server-rendered, light mode only)
    cy.visit('/admin/login');
    cy.wait(1000);
    cy.screenshot('login-page-light', { overwrite: true });

    // Login to access the Vue admin dashboard
    const username = Cypress.env('LISTMONK_ADMIN_USER') || 'admin';
    const password = Cypress.env('LISTMONK_ADMIN_PASSWORD') || 'listmonk';

    cy.get('input[name=username]').type(username);
    cy.get('input[name=password]').type(password);
    cy.get('button[type=submit]').click();

    // Wait for the admin app to load
    cy.wait(3000);

    // Navigate to dashboard explicitly
    cy.visit('/admin');
    cy.wait(2000);

    // Take dashboard screenshot in light mode
    cy.screenshot('dashboard-light', { overwrite: true });

    // Open the user dropdown to show the toggle
    cy.get('.navbar-end .user').click();

    // Wait for dropdown menu items to be visible
    cy.contains('Dark Mode').should('be.visible');
    cy.wait(800);

    // Take full page screenshot showing the dropdown menu
    cy.screenshot('dark-mode-toggle-menu', {
      overwrite: true,
      capture: 'fullPage'
    });

    // Click the dark mode toggle
    cy.contains('Dark Mode').click();
    cy.wait(1500);

    // Take dashboard screenshot in dark mode
    cy.screenshot('dashboard-dark', { overwrite: true });
  });
});
