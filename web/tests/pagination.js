async (page) => {
  await page.keyboard.press('Escape');
  const size = page.getByRole('combobox', { name: '每页', exact: true });
  await page.getByRole('button', { name: '下一页', exact: true }).click();
  await page.getByRole('button', { name: 'db-20', exact: true }).waitFor();
  await size.click();
  await page.getByRole('option', { name: '50', exact: true }).click();
  await page.getByRole('button', { name: 'db-49', exact: true }).waitFor();
  if (!page.url().includes('page_size=50') || !page.url().includes('page=1'))
    throw Error('切换条数未重置页码');
  await size.focus();
  await page.keyboard.press('ArrowDown');
  await page.getByRole('option', { name: '50', exact: true }).waitFor();
  await page.waitForFunction(() => document.activeElement?.getAttribute('role') === 'option');
  await page.keyboard.press('Home');
  await page.waitForFunction(() => document.activeElement?.textContent === '10');
  await page.keyboard.press('Enter');
  await page.waitForFunction(() => document.querySelectorAll('tbody tr').length === 10);
  if ((await page.locator('tbody tr').count()) !== 10) throw Error('键盘选择条数未生效');
  await size.click();
  await page.keyboard.press('Escape');
  await page.waitForFunction(() => document.activeElement?.classList.contains('pagination-select'));
  if (!(await size.evaluate((element) => element === document.activeElement)))
    throw Error('焦点未恢复');
  for (const title of ['Database', 'Linux', 'Kubernetes', 'Audit']) {
    await page.getByRole('link', { name: title, exact: true }).click();
    await size.click();
    if ((await page.getByRole('option').count()) !== 4) throw Error('分页选项不一致');
    await page.keyboard.press('Escape');
  }
  await page.setViewportSize({ width: 1440, height: 900 });
  await size.click();
  await page.screenshot({ path: 'output/playwright/pagination-desktop.png' });
  await page.keyboard.press('Escape');
  await page.getByRole('button', { name: '切换主题' }).click();
  await page.setViewportSize({ width: 390, height: 844 });
  await size.click();
  const box = await page.getByRole('listbox').boundingBox();
  if (!box || box.x < 0 || box.x + box.width > 390 || box.y < 0 || box.y + box.height > 844)
    throw Error('弹层越界');
  await page.screenshot({ path: 'output/playwright/pagination-mobile-dark.png' });
  await page.keyboard.press('Escape');
  await page.getByRole('button', { name: '切换主题' }).click();
}
