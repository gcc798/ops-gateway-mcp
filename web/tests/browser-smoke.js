async (page) => {
  const check = (value, message) => {
    if (!value) throw new Error(message);
  };
  await page.getByRole('button', { name: '名称', exact: true }).click();
  await page.getByRole('button', { name: 'db-64', exact: true }).waitFor();
  await page.getByRole('button', { name: '下一页', exact: true }).click();
  await page.getByRole('button', { name: 'db-44', exact: true }).waitFor();
  await page.getByRole('button', { name: 'db-44', exact: true }).click();
  await page.reload();
  await page.getByRole('heading', { name: 'db-44', exact: true }).waitFor();
  check(page.url().includes('page=2'), '刷新详情丢失页码');
  await page.getByRole('button', { name: '返回资源列表' }).click();
  await page.getByRole('button', { name: 'db-44', exact: true }).waitFor();
  await page.getByRole('searchbox', { name: '名称', exact: true }).fill('db-00');
  await page.getByRole('button', { name: '查询', exact: true }).click();
  await page.getByRole('button', { name: 'db-00', exact: true }).waitFor();
  check((await page.locator('tbody tr').count()) === 1, '名称筛选未生效');
  await page.goBack();
  await page.getByRole('button', { name: 'db-44', exact: true }).waitFor();
  await page.goForward();
  await page.getByRole('button', { name: 'db-00', exact: true }).click();
  await page.getByRole('link', { name: '相关审计' }).click();
  check(page.url().includes('resource=db-00'), '关联审计缺少资源筛选');
  await page.getByRole('button', { name: '待确认', exact: true }).click();
  await page.getByRole('button', { name: '详情', exact: true }).first().waitFor();
  await page.getByRole('button', { name: '详情', exact: true }).first().click();
  await page.getByRole('button', { name: '确认执行', exact: true }).waitFor();
  if (await page.getByRole('button', { name: '确认执行', exact: true }).isDisabled()) {
    await page.getByRole('button', { name: 'Audit', exact: true }).click();
    await page.getByRole('button', { name: '详情', exact: true }).last().click();
  }
  await page.getByRole('button', { name: '确认执行', exact: true }).click();
  await page.getByRole('alert').waitFor();
  await page.getByText('failed', { exact: true }).waitFor();
  check(
    (await page.getByRole('button', { name: '确认执行', exact: true }).count()) === 0,
    '失败后仍显示重复执行',
  );
  await page.getByRole('link', { name: 'Linux', exact: true }).click();
  await page.getByRole('button', { name: 'dev-host', exact: true }).click();
  await page.getByText('fixture-password', { exact: true }).waitFor();
  await page.getByRole('button', { name: '复制', exact: true }).first().click();
  await page.getByRole('button', { name: '已复制', exact: true }).waitFor();
  await page.getByRole('link', { name: 'Kubernetes', exact: true }).click();
  await page.getByRole('button', { name: 'dev-cluster', exact: true }).click();
  await page.getByText('fixture-token', { exact: true }).waitFor();
  await page.getByRole('link', { name: 'Policy', exact: true }).click();
  await page.getByText('滚动重启', { exact: true }).waitFor();
  await page.getByRole('link', { name: 'Settings', exact: true }).click();
  await page.getByRole('checkbox', { name: '深色主题' }).check();
  await page.reload();
  await page.getByRole('checkbox', { name: '深色主题' }).waitFor();
  check(await page.getByRole('checkbox', { name: '深色主题' }).isChecked(), '刷新丢失主题');
  await page.getByRole('checkbox', { name: '深色主题' }).uncheck();
  await page.getByRole('link', { name: 'Database', exact: true }).click();
  await page.getByRole('button', { name: 'db-00', exact: true }).waitFor();
  await page.setViewportSize({ width: 1440, height: 1000 });
  await page.screenshot({ path: 'output/playwright/database-desktop.png', fullPage: true });
  await page.setViewportSize({ width: 390, height: 844 });
  check(
    await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth),
    '移动端页面横向溢出',
  );
  await page.screenshot({ path: 'output/playwright/database-mobile.png', fullPage: true });
  await page.getByRole('link', { name: 'Audit', exact: true }).click();
  await page.getByRole('button', { name: '待确认', exact: true }).click();
  await page.getByRole('button', { name: '详情', exact: true }).first().click();
  await page.getByRole('button', { name: '确认执行', exact: true }).waitFor();
  check(
    await page.getByRole('button', { name: '确认执行', exact: true }).isDisabled(),
    '过期确认未禁用',
  );
  await page.screenshot({ path: 'output/playwright/audit-mobile.png', fullPage: true });
  console.log(
    'PASS: 排序、分页、详情刷新、前进后退、筛选、关联审计、真实确认失败、过期、明文复制、主题及移动端',
  );
};
