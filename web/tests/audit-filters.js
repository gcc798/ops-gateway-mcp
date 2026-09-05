async (page) => {
  await page.getByRole('textbox', { name: '开始时间（含）' }).fill('2026-09-01 10:20:31');
  await page.keyboard.press('Tab');
  await page
    .getByRole('combobox', { name: '工具 / REST 路由', exact: true })
    .selectOption('db_prepare_execute');
  await page
    .getByRole('combobox', { name: 'Token 身份', exact: true })
    .selectOption('browser-test');
  await page.getByRole('button', { name: '查询', exact: true }).click();
  await page.reload();
  await page.getByRole('combobox', { name: 'Token 身份', exact: true }).waitFor();
  if (
    (await page.getByRole('textbox', { name: '开始时间（含）' }).inputValue()) !==
    '2026-09-01 10:20:31'
  )
    throw Error('日期刷新失败');
  if (
    (await page.getByRole('combobox', { name: 'Token 身份', exact: true }).inputValue()) !==
    'browser-test'
  )
    throw Error('身份丢失');
  await page.getByRole('button', { name: '重置', exact: true }).click();
  if (await page.getByRole('textbox', { name: '开始时间（含）' }).inputValue())
    throw Error('重置失败');
};
