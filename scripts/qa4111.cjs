// Complete browser -> protected API -> response flow using isolated Go fixtures.
const {chromium}=require('playwright');
(async()=>{
 const browser=await chromium.launch({executablePath:process.env.ARIA_TEST_BROWSER,headless:true,args:['--no-sandbox','--no-proxy-server']});
 try{
  const page=await browser.newPage({viewport:{width:390,height:844}});const errors=[],exports=[];
  page.on('pageerror',e=>errors.push(e.message));page.on('request',r=>{if(r.url().includes('/api/edges/export'))exports.push(r.url())});
  await page.goto(process.argv[2]);await page.locator('#login-form [name=username]').fill('admin');await page.locator('#login-form [name=password]').fill('Test-only-password-123');await page.locator('#login-form button[type=submit]').click();await page.locator('#app-screen').waitFor({state:'visible'});
  await page.waitForFunction(()=>document.querySelector('.control-country img.country-flag')?.naturalWidth>0);
  const image=await page.locator('.control-country img.country-flag').first().evaluate(e=>({width:e.naturalWidth,src:e.getAttribute('src').slice(0,26)}));
  await page.evaluate(()=>{const node=document.querySelector('.control-country');node.innerHTML=geoBadge({code:'FR',country:'France',ip:'8.8.8.8',flagImage:'data:image/svg+xml;base64,'})});
  if(await page.locator('.control-country img').count()||await page.locator('.control-country .country-flag-emoji').textContent()!=='🇫🇷')throw Error('empty flag payload produced a broken image');
  await page.evaluate(()=>{const node=document.querySelector('.control-country');node.innerHTML=geoBadge({code:'FR',country:'France',ip:'8.8.8.8',flagImage:'data:image/svg+xml;base64,YmFk'});});
  await page.locator('.control-country .country-flag-emoji').waitFor();
  await page.locator('.mobile-tools [data-view=hosting]').click();await page.locator('[data-control-view=edges]').click();
  const trial=page.getByRole('button',{name:'خروجی آزمایشی برای تست دستی',exact:true});await trial.waitFor();if(!await trial.isEnabled())throw Error('trial output is disabled despite fresh TLS/origin proof');await trial.click();
  await page.locator('#control-form button[type=submit]').click();await page.locator('#relay-output-dialog').waitFor({state:'visible'});
  if(!exports.some(u=>u.includes('trial=1'))||!(await page.locator('#relay-output-content textarea').inputValue()).includes('trial.workers.dev'))throw Error('trial did not use selected Worker');
  if(await page.locator('#relay-output-content').getByRole('button',{name:'تحویل متن و QR در تلگرام',exact:true}).count())throw Error('trial exposes automatic Telegram delivery');
  if(!await page.locator('#relay-output-content').getByText('خروجی آزمایشی برای بررسی دستی',{exact:false}).count())throw Error('trial warning missing');
  if(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth))throw Error('mobile overflow');if(errors.length)throw Error(errors.join('\n'));
  console.log(JSON.stringify({flagImage:image,emptyFlagFallback:true,brokenFlagFallback:true,trialAPI:true,selectedWorkerLink:true,automaticDeliveryHidden:true,mobileOverflow:false,errors}));
 }finally{await browser.close()}
})().catch(e=>{console.error(e.stack);process.exitCode=1});
