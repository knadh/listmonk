// Use Playwright's bundled Chromium unless an executable is explicitly supplied.
export const launchOptions = {
  executablePath: process.env.CHROMIUM_PATH || undefined,
};
