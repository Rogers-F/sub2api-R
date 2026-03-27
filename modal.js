/* 功能1: 公告弹窗 - 在第二个h2.text-lg后添加"查看全部"链接 */
var ob=new MutationObserver(function(){if(!localStorage.getItem("auth_token"))return;if(document.getElementById("viewall"))return;var a=document.querySelectorAll("h2.text-lg")[1];if(!a)return;var b=document.createElement("a");b.id="viewall";b.textContent="查看全部 >";b.style="color:#3b82f6;cursor:pointer;font-size:14px;margin-left:10px";b.onclick=function(){var m=document.createElement("div");m.style="position:fixed;top:0;left:0;width:100vw;height:100vh;background:rgba(0,0,0,0.5);z-index:9999;display:flex;align-items:center;justify-content:center";m.onclick=function(e){if(e.target===m)m.remove()};var d=document.createElement("div");d.style="background:white;width:95%;max-width:1000px;height:90vh;border-radius:12px;overflow:auto;padding:30px;position:relative";d.innerHTML="<div id=ann-app>加载中...</div>";m.appendChild(d);document.body.appendChild(m);var sc=document.createElement("script");sc.src="/custom-js/ann.js";document.body.appendChild(sc)};a.after(b)});ob.observe(document.body,{childList:true,subtree:true});

/* 功能2: 推荐奖励页-奖励规则描述 */
new MutationObserver(function(){if(location.pathname!=="/referral")return;if(document.getElementById("ref-desc"))return;var els=document.querySelectorAll("h2.text-lg");for(var i=0;i<els.length;i++){if(els[i].textContent.trim()==="奖励规则"){var s=document.createElement("span");s.id="ref-desc";s.style="font-size:12px;color:#6b7280;margin-left:8px;font-weight:normal";s.textContent="邀请好友订阅：双方各得$10（每次） | PAYG返佣：好友充值后你得1%（每次）";els[i].appendChild(s)}}}).observe(document.body,{childList:true,subtree:true});

/* 功能3: 加载导航栏脚本 */
var ns=document.createElement("script");ns.src="/custom-js/nav.js";document.body.appendChild(ns);

/* 功能5: 加载PAYG钱包脚本 */
var pg=document.createElement("script");pg.src="/custom-js/payg.js";document.body.appendChild(pg);

/* 功能6: 加载管理员推广统计脚本 */
var ra=document.createElement("script");ra.src="/custom-js/referral-admin.js";document.body.appendChild(ra);

/* 功能4: 文字替换 */
setInterval(function(){var a=document.querySelectorAll("span,p,div,h1,h2,h3");for(var i=0;i<a.length;i++){if(a[i].children.length===0){if(a[i].textContent.indexOf("注册奖励")>=0){a[i].textContent=a[i].textContent.replace(/注册奖励/g,"订阅奖励")}if(a[i].textContent.indexOf("好友成功注册")>=0){a[i].textContent=a[i].textContent.replace(/好友成功注册/g,"好友成功订阅")}if(a[i].textContent.indexOf("邀请好友注册")>=0){a[i].textContent=a[i].textContent.replace(/邀请好友注册/g,"邀请好友订阅")}if(location.pathname==="/referral"&&a[i].textContent==="+$0.00"){a[i].textContent="+$10.00"}if(a[i].textContent.trim()==="客服功能"){a[i].textContent="微信号：w414515660"}}}},200);
