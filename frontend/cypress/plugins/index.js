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
      const smtpSQL = 'UPDATE settings SET value = \'[{"host":"localhost","name":"email-mailhog","port":1025,"uuid":"","enabled":true,"password":"","tls_type":"none","username":"","max_conns":10,"idle_timeout":"15s","wait_timeout":"5s","auth_protocol":"none","email_headers":[],"hello_hostname":"","max_msg_retries":2,"tls_skip_verify":true},{"host":"smtp.gmail.com","port":465,"enabled":false,"password":"password","tls_type":"TLS","username":"username@gmail.com","max_conns":10,"idle_timeout":"15s","wait_timeout":"5s","auth_protocol":"login","email_headers":[],"hello_hostname":"","max_msg_retries":2,"tls_skip_verify":false}]\' WHERE key = \'smtp\';';
      try {
        execSync('docker exec -i listmonk_db psql -U listmonk -d listmonk_test', {
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
