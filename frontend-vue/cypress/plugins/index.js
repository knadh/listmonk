/// <reference types="cypress" />

const { execSync, spawn } = require('child_process');
const path = require('path');

/**
 * @type {Cypress.PluginConfig}
 */
module.exports = (on, config) => {
  const rootDir = path.resolve(__dirname, '..', '..', '..');

  on('task', {
    // Kill listmonk, reset the DB, and start the server in the background.
    resetServer({ blank = false } = {}) {
      try {
        execSync('pkill -9 listmonk', { stdio: 'ignore' });
      } catch (e) {
        // Do nothing.
      }

      // Run install.
      const env = blank
        ? { ...process.env }
        : { ...process.env, LISTMONK_ADMIN_USER: 'admin', LISTMONK_ADMIN_PASSWORD: 'listmonk' };

      execSync('./listmonk --install --yes', { cwd: rootDir, env, stdio: 'ignore' });

      // Replace the first SMTP block with local MailHog.
      const smtpSQL = "UPDATE settings SET value = (SELECT jsonb_agg(smtp || jsonb_build_object('host','localhost','port',1025,'tls_type','none')) FROM jsonb_array_elements(value) AS smtp) WHERE key = 'smtp';";
      try {
        execSync('docker exec -i listmonk_db psql -U listmonk -d listmonk', {
          input: smtpSQL,
          stdio: ['pipe', 'ignore', 'ignore'],
        });
      } catch (e) {
        // Do nothing.
      }

      // Start the server.
      const child = spawn('./listmonk', [], {
        cwd: rootDir,
        detached: true,
        stdio: 'ignore',
      });
      child.unref();

      return null;
    },
  });
};
