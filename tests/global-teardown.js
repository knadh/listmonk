import { execSync } from 'node:child_process';

export default function globalTeardown() {
  try { execSync('pkill -9 listmonk', { stdio: 'ignore' }); } catch { /* none running */ }
}
