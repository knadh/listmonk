// Open a table row's "More" actions dropdown.
export async function openMenu(row) {
  await row.getByRole('button', { name: 'More' }).click();
}

// Click the "Ok" button on the confirmation dialog.
export async function confirm(page) {
  await page.locator('dialog[open] [data-confirm-ok]').click();
}
