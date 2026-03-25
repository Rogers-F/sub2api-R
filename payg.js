/* PAYG钱包 - 侧边栏注入 + 钱包页面 */
new MutationObserver(function(){
if(document.getElementById("nav-payg"))return;
var ref=document.querySelector('a[href="/referral"]');
if(!ref)return;

var a=document.createElement("a");
a.id="nav-payg";
a.href="javascript:void(0)";
a.className="sidebar-link mb-1";
a.title="PAYG钱包";

/* 钱包图标 Heroicons outline */
var svg=document.createElementNS("http://www.w3.org/2000/svg","svg");
svg.setAttribute("class","h-5 w-5 flex-shrink-0");
svg.setAttribute("fill","none");
svg.setAttribute("viewBox","0 0 24 24");
svg.setAttribute("stroke","currentColor");
svg.setAttribute("stroke-width","1.5");
var p=document.createElementNS("http://www.w3.org/2000/svg","path");
p.setAttribute("stroke-linecap","round");
p.setAttribute("stroke-linejoin","round");
p.setAttribute("d","M21 12a2.25 2.25 0 0 0-2.25-2.25H15a3 3 0 1 1-6 0H5.25A2.25 2.25 0 0 0 3 12m18 0v6a2.25 2.25 0 0 1-2.25 2.25H5.25A2.25 2.25 0 0 1 3 18v-6m18 0V9M3 12V9m18 0a2.25 2.25 0 0 0-2.25-2.25H5.25A2.25 2.25 0 0 0 3 9m18 0V6a2.25 2.25 0 0 0-2.25-2.25H5.25A2.25 2.25 0 0 0 3 6v3");
svg.appendChild(p);
a.appendChild(svg);

var sp=document.createElement("span");
sp.textContent="PAYG钱包";
a.appendChild(sp);

a.onclick=function(){
  var all=document.querySelectorAll(".sidebar-link");
  for(var i=0;i<all.length;i++){
    all[i].classList.remove("sidebar-link-active","router-link-active","router-link-exact-active");
  }
  a.classList.add("sidebar-link-active");
  history.replaceState(null,null,location.pathname+location.search+"#payg-wallet");

  var main=document.querySelector("main");
  if(!main)main=document.querySelector(".flex-1");
  if(!main)return;

  main.innerHTML='<div style="max-width:1200px;margin:0 auto;padding:16px;">'
  +'<div style="display:flex;align-items:baseline;justify-content:space-between;flex-wrap:wrap;gap:8px;">'
    +'<div><h1 style="font-size:20px;font-weight:700;margin:0;">账户资金</h1>'
    +'<p style="opacity:0.6;font-size:13px;margin-top:4px;">查看您的余额概况与充值记录</p></div>'
    +'<div style="text-align:right;"><span style="opacity:0.6;font-size:13px;">钱包余额</span><br><span id="payg-balance-top" style="font-size:18px;font-weight:700;">加载中...</span></div>'
  +'</div>'

  +'<div style="display:grid;grid-template-columns:repeat(auto-fit,minmax(200px,1fr));gap:12px;margin-top:16px;">'
    +'<div style="background:#f0fdf4;border-radius:12px;padding:16px;">'
      +'<div style="color:#6b7280;font-size:13px;">可用总余额 (USD)</div>'
      +'<div id="payg-balance" style="font-size:24px;font-weight:700;color:#111827;margin-top:6px;">加载中...</div>'
    +'</div>'
    +'<div style="background:#eff6ff;border-radius:12px;padding:16px;">'
      +'<div style="color:#6b7280;font-size:13px;">累计充值金额 (USD)</div>'
      +'<div id="payg-recharge" style="font-size:24px;font-weight:700;color:#111827;margin-top:6px;">加载中...</div>'
    +'</div>'
    +'<div style="background:#fefce8;border-radius:12px;padding:16px;">'
      +'<div style="color:#6b7280;font-size:13px;">累计消费金额 (USD)</div>'
      +'<div id="payg-consumption" style="font-size:24px;font-weight:700;color:#111827;margin-top:6px;">加载中...</div>'
    +'</div>'
  +'</div>'

  +'<div style="background:white;color:#111827;border:1px solid #e5e7eb;border-radius:12px;padding:16px;margin-top:16px;">'
    +'<h2 style="font-size:16px;font-weight:600;margin:0;">在线充值 <span style="font-size:12px;font-weight:400;color:#9ca3af;">选择充值金额(CNY) - 自动兑换为USD余额</span></h2>'

    +'<div style="margin-top:12px;">'
      +'<div style="color:#374151;font-size:13px;font-weight:500;margin-bottom:8px;">充值金额</div>'
      +'<div style="display:grid;grid-template-columns:repeat(3,1fr);gap:10px;">'
        +'<button class="payg-amt" style="padding:10px 0 8px;border:2px solid #e5e7eb;border-radius:8px;background:white;cursor:pointer;font-size:15px;font-weight:600;color:#374151;text-align:center;" data-val="50">\u00a550<br><span style="font-size:11px;font-weight:400;color:#9ca3af;">到账 $50.00</span></button>'
        +'<button class="payg-amt" style="padding:10px 0 8px;border:2px solid #e5e7eb;border-radius:8px;background:white;cursor:pointer;font-size:15px;font-weight:600;color:#374151;text-align:center;" data-val="80">\u00a580<br><span style="font-size:11px;font-weight:400;color:#9ca3af;">到账 $80.00</span></button>'
        +'<button class="payg-amt" style="padding:10px 0 8px;border:2px solid #e5e7eb;border-radius:8px;background:white;cursor:pointer;font-size:15px;font-weight:600;color:#374151;text-align:center;" data-val="100">\u00a5100<br><span style="font-size:11px;font-weight:400;color:#9ca3af;">到账 $100.00</span></button>'
      +'</div>'
      +'<input id="payg-custom" type="number" placeholder="自定义金额" style="margin-top:10px;padding:12px 16px;border:2px solid #e5e7eb;border-radius:8px;font-size:15px;width:200px;box-sizing:border-box;outline:none;color:#111827;background:white;">'
    +'</div>'

    +'<button id="payg-submit" style="margin-top:14px;padding:10px 40px;background:#3b82f6;color:white;border:none;border-radius:8px;font-size:14px;font-weight:600;cursor:pointer;">立即支付</button>'
  +'</div>'

  +'<div style="background:white;color:#111827;border:1px solid #e5e7eb;border-radius:12px;padding:16px;margin-top:16px;">'
    +'<div style="display:flex;border-bottom:2px solid #e5e7eb;">'
      +'<div class="payg-tab payg-tab-active" data-tab="recharge" style="padding:10px 16px;font-weight:600;color:#3b82f6;border-bottom:2px solid #3b82f6;margin-bottom:-2px;cursor:pointer;font-size:14px;">充值记录</div>'
      +'<div class="payg-tab" data-tab="usage" style="padding:10px 16px;font-weight:500;color:#6b7280;cursor:pointer;font-size:14px;">消费日志</div>'
    +'</div>'
    +'<div style="display:flex;gap:8px;margin-top:12px;flex-wrap:wrap;align-items:center;">'
      +'<input id="payg-search" type="text" placeholder="搜索订单号..." style="padding:8px 12px;border:1px solid #e5e7eb;border-radius:6px;font-size:13px;outline:none;color:#111827;background:white;flex:1;min-width:120px;">'
      +'<select id="payg-filter" style="padding:8px 12px;border:1px solid #e5e7eb;border-radius:6px;font-size:13px;color:#374151;background:white;outline:none;cursor:pointer;">'
        +'<option value="">全部类型</option>'
        +'<option value="PAID">已支付</option>'
      +'</select>'
      +'<select id="payg-limit" style="padding:8px 12px;border:1px solid #e5e7eb;border-radius:6px;font-size:13px;color:#374151;background:white;outline:none;cursor:pointer;">'
        +'<option value="20">20条</option>'
        +'<option value="50">50条</option>'
        +'<option value="100">100条</option>'
      +'</select>'
    +'</div>'
    +'<div id="payg-orders" style="margin-top:12px;"><div style="text-align:center;padding:40px 0;color:#9ca3af;font-size:13px;">加载中...</div></div>'
  +'</div>'
  +'</div>';

  /* 获取用户信息并判断角色 */
  var paygUserId=null;
  var token=localStorage.getItem("auth_token")||"";
  var authUser=null;
  try{authUser=JSON.parse(localStorage.getItem("auth_user")||"{}")}catch(e){}
  var isAdmin=authUser&&(authUser.role==="admin"||authUser.email==="xingsuancode@qq.com");

  if(isAdmin){
    /* ========== 管理员视图 ========== */
    main.innerHTML='<div style="max-width:1200px;margin:0 auto;padding:16px;">'
    +'<div style="display:flex;align-items:baseline;justify-content:space-between;flex-wrap:wrap;gap:8px;">'
      +'<div><h1 style="font-size:20px;font-weight:700;margin:0;">PAYG充值管理</h1>'
      +'<p style="opacity:0.6;font-size:13px;margin-top:4px;">全平台用户充值统计与订单管理</p></div>'
      +'<div style="text-align:right;"><span style="opacity:0.6;font-size:13px;">我的钱包余额</span><br><span id="admin-my-balance" style="font-size:18px;font-weight:700;">加载中...</span></div>'
    +'</div>'
    +'<div style="display:grid;grid-template-columns:repeat(2,1fr);gap:12px;margin-top:16px;">'
      +'<div style="background:#f0fdf4;border-radius:12px;padding:16px;">'
        +'<div style="color:#6b7280;font-size:13px;">平台总充值 (CNY)</div>'
        +'<div id="admin-total" style="font-size:24px;font-weight:700;color:#111827;margin-top:6px;">加载中...</div>'
      +'</div>'
      +'<div style="background:#eff6ff;border-radius:12px;padding:16px;">'
        +'<div style="color:#6b7280;font-size:13px;">已支付订单</div>'
        +'<div id="admin-paid" style="font-size:24px;font-weight:700;color:#111827;margin-top:6px;">加载中...</div>'
      +'</div>'
    +'</div>'
    +'<div style="background:white;color:#111827;border:1px solid #e5e7eb;border-radius:12px;padding:16px;margin-top:16px;">'
      +'<h2 style="font-size:16px;font-weight:600;margin:0;">用户充值汇总</h2>'
      +'<p style="color:#6b7280;font-size:13px;margin-top:4px;">点击用户行查看该用户的订单详情</p>'
      +'<div id="admin-users" style="margin-top:12px;"><div style="text-align:center;padding:30px 0;color:#9ca3af;font-size:13px;">加载中...</div></div>'
    +'</div>'
    +'<div style="background:white;color:#111827;border:1px solid #e5e7eb;border-radius:12px;padding:16px;margin-top:16px;">'
      +'<h2 style="font-size:16px;font-weight:600;margin:0;">最近订单</h2>'
      +'<div style="display:flex;gap:8px;margin-top:12px;flex-wrap:wrap;align-items:center;">'
        +'<input id="admin-search" type="text" placeholder="搜索用户/订单号..." style="padding:8px 12px;border:1px solid #e5e7eb;border-radius:6px;font-size:13px;outline:none;color:#111827;background:white;flex:1;min-width:120px;">'
        +'<select id="admin-filter" style="padding:8px 12px;border:1px solid #e5e7eb;border-radius:6px;font-size:13px;color:#374151;background:white;outline:none;cursor:pointer;">'
          +'<option value="">全部类型</option>'
          +'<option value="PAID">已支付</option>'
        +'</select>'
        +'<select id="admin-limit" style="padding:8px 12px;border:1px solid #e5e7eb;border-radius:6px;font-size:13px;color:#374151;background:white;outline:none;cursor:pointer;">'
          +'<option value="20">20条</option>'
          +'<option value="50">50条</option>'
          +'<option value="100">100条</option>'
        +'</select>'
      +'</div>'
      +'<div id="admin-orders" style="margin-top:12px;"><div style="text-align:center;padding:30px 0;color:#9ca3af;font-size:13px;">加载中...</div></div>'
    +'</div>'
    +'</div>';
    /* 加载管理员数据 */
    /* 加载管理员自己的钱包余额 */
    fetch("/api/v1/user/profile",{headers:{"Authorization":"Bearer "+token}})
    .then(function(r){return r.json()}).then(function(d){
      if(d&&d.data&&d.data.id){
        fetch("/api/pay/wallet?user_id="+d.data.id).then(function(r){return r.json()}).then(function(w){
          if(w.success){document.getElementById("admin-my-balance").textContent="$"+w.balance.toFixed(2);}
        });
      }
    }).catch(function(){});
    fetch("/api/pay/admin/wallet").then(function(r){return r.json()}).then(function(d){
      if(!d.success)return;
      document.getElementById("admin-total").textContent="\u00a5"+d.total_recharge.toFixed(2);
      document.getElementById("admin-paid").textContent=d.paid_orders;
      /* 用户汇总表 */
      var uel=document.getElementById("admin-users");
      if(!d.users||!d.users.length){uel.innerHTML='<div style="text-align:center;padding:40px 0;color:#9ca3af;">暂无数据</div>';return;}
      var uh='<div style="overflow-x:auto;"><table style="width:100%;border-collapse:collapse;font-size:13px;">'
        +'<thead><tr style="text-align:left;color:#6b7280;">'
        +'<th style="padding:10px 12px;border-bottom:1px solid #e5e7eb;">用户邮箱</th>'
        +'<th style="padding:10px 12px;border-bottom:1px solid #e5e7eb;">充值总额 (CNY)</th>'
        +'<th style="padding:10px 12px;border-bottom:1px solid #e5e7eb;">订单数</th>'
        +'</tr></thead><tbody>';
      for(var i=0;i<d.users.length;i++){
        var u=d.users[i];
        uh+='<tr class="admin-user-row" data-uid="'+(u.user_id||"")+'" style="border-bottom:1px solid #f3f4f6;cursor:pointer;" onmouseover="this.style.background=\'#f9fafb\'" onmouseout="this.style.background=\'white\'">'
          +'<td style="padding:10px 12px;color:#111827;font-weight:500;">'+u.email+'</td>'
          +'<td style="padding:10px 12px;font-weight:600;">\u00a5'+u.total_recharge.toFixed(2)+'</td>'
          +'<td style="padding:10px 12px;color:#6b7280;">'+u.order_count+'</td>'
          +'</tr>'
          +'<tr class="admin-user-detail" data-uid="'+(u.user_id||"")+'" style="display:none;"><td colspan="3" style="padding:0 12px 12px;"></td></tr>';
      }
      uh+='</tbody></table></div>';
      uel.innerHTML=uh;
      /* 点击展开用户订单详情 */
      var rows=document.querySelectorAll(".admin-user-row");
      for(var j=0;j<rows.length;j++){
        rows[j].onclick=(function(row){return function(){
          var uid=row.getAttribute("data-uid");
          var detail=document.querySelector('.admin-user-detail[data-uid="'+uid+'"]');
          if(!detail)return;
          if(detail.style.display==="none"){
            detail.style.display="table-row";
            var cell=detail.querySelector("td");
            var userOrders=[];
            for(var k=0;k<d.orders.length;k++){
              if(String(d.orders[k].user_id)===uid||(uid===""&&d.orders[k].user_id===null))userOrders.push(d.orders[k]);
            }
            if(!userOrders.length){cell.innerHTML='<div style="padding:12px;color:#9ca3af;font-size:13px;">该用户无订单记录</div>';return;}
            var oh='<table style="width:100%;border-collapse:collapse;font-size:12px;background:#f9fafb;border-radius:8px;white-space:nowrap;">'
              +'<thead><tr style="color:#9ca3af;"><th style="padding:8px;">时间</th><th style="padding:8px;">金额</th><th style="padding:8px;">状态</th><th style="padding:8px;">订单号</th></tr></thead><tbody>';
            for(var k=0;k<userOrders.length;k++){
              var o=userOrders[k];
              var t=o.created_at?o.created_at.substring(0,19).replace("T"," "):"";
              var st=o.status==="PAID"?"已支付":"待支付";
              var sc=o.status==="PAID"?"#10b981":"#f59e0b";
              oh+='<tr><td style="padding:6px 8px;color:#6b7280;">'+t+'</td><td style="padding:6px 8px;font-weight:600;">\u00a5'+o.amount_yuan.toFixed(2)+'</td><td style="padding:6px 8px;"><span style="color:'+sc+';">'+st+'</span></td><td style="padding:6px 8px;color:#9ca3af;font-size:11px;">'+o.sn+'</td></tr>';
            }
            oh+='</tbody></table>';
            cell.innerHTML=oh;
          }else{
            detail.style.display="none";
          }
        }})(rows[j]);
      }
      /* 全部最近订单 */
      var _adminAllOrders=d.orders||[];
      function _adminFilterOrders(){
        var oel=document.getElementById("admin-orders");
        var sf=document.getElementById("admin-search");
        var ff=document.getElementById("admin-filter");
        var lf=document.getElementById("admin-limit");
        var kw=sf?sf.value.trim().toLowerCase():"";
        var st=ff?ff.value:"";
        var lim=lf?parseInt(lf.value):20;
        var filtered=[];
        for(var k=0;k<_adminAllOrders.length;k++){
          var o=_adminAllOrders[k];
          if(st&&o.status!==st)continue;
          if(kw&&(o.sn||"").toLowerCase().indexOf(kw)===-1&&(o.email||"").toLowerCase().indexOf(kw)===-1)continue;
          filtered.push(o);
        }
        var show=filtered.slice(0,lim);
        if(!show.length){oel.innerHTML='<div style="text-align:center;padding:30px 0;color:#9ca3af;font-size:13px;">暂无匹配订单</div>';return;}
        var oh2='<div style="overflow-x:auto;-webkit-overflow-scrolling:touch;"><table style="width:100%;border-collapse:collapse;font-size:13px;white-space:nowrap;">'
          +'<thead><tr style="text-align:left;color:#6b7280;">'
          +'<th style="padding:10px 12px;border-bottom:1px solid #e5e7eb;">时间</th>'
          +'<th style="padding:10px 12px;border-bottom:1px solid #e5e7eb;">用户</th>'
          +'<th style="padding:10px 12px;border-bottom:1px solid #e5e7eb;">金额</th>'
          +'<th style="padding:10px 12px;border-bottom:1px solid #e5e7eb;">状态</th>'
          +'<th style="padding:10px 12px;border-bottom:1px solid #e5e7eb;">订单号</th>'
          +'</tr></thead><tbody>';
        for(var i=0;i<show.length;i++){
          var o=show[i];
          var t=o.created_at?o.created_at.substring(0,19).replace("T"," "):"";
          var sc2=o.status==="PAID"?"#10b981":"#f59e0b";
          var st2=o.status==="PAID"?"已支付":"待支付";
          oh2+='<tr style="border-bottom:1px solid #f3f4f6;">'
            +'<td style="padding:10px 12px;color:#6b7280;">'+t+'</td>'
            +'<td style="padding:10px 12px;color:#111827;">'+o.email+'</td>'
            +'<td style="padding:10px 12px;font-weight:600;">\u00a5'+o.amount_yuan.toFixed(2)+'</td>'
            +'<td style="padding:10px 12px;"><span style="color:'+sc2+';font-weight:500;">'+st2+'</span></td>'
            +'<td style="padding:10px 12px;color:#9ca3af;font-size:12px;">'+o.sn+'</td>'
            +'</tr>';
        }
        oh2+='</tbody></table></div>';
        if(filtered.length>lim)oh2+='<div style="text-align:center;padding:8px;color:#9ca3af;font-size:12px;">显示前'+lim+'条，共'+filtered.length+'条</div>';
        oel.innerHTML=oh2;
      }
      _adminFilterOrders();
      var _asf=document.getElementById("admin-search");
      var _aff=document.getElementById("admin-filter");
      var _alf=document.getElementById("admin-limit");
      if(_asf)_asf.oninput=function(){_adminFilterOrders();};
      if(_aff)_aff.onchange=function(){_adminFilterOrders();};
      if(_alf)_alf.onchange=function(){_adminFilterOrders();};
    }).catch(function(){});
  }else{
    /* ========== 普通用户视图 ========== */
    fetch("/api/v1/user/profile",{headers:{"Authorization":"Bearer "+token}})
    .then(function(r){return r.json()})
    .then(function(d){
      if(d&&d.data&&d.data.id){
        paygUserId=d.data.id;
        loadWallet(paygUserId);
      }else{
        document.getElementById("payg-balance").textContent="$0.00";
        var bt1=document.getElementById("payg-balance-top");if(bt1)bt1.textContent="$0.00";
        document.getElementById("payg-recharge").textContent="$0.00";
        document.getElementById("payg-consumption").textContent="$0.00";
      }
    }).catch(function(){
      document.getElementById("payg-balance").textContent="$0.00";
      var bt2=document.getElementById("payg-balance-top");if(bt2)bt2.textContent="$0.00";
      document.getElementById("payg-recharge").textContent="$0.00";
      document.getElementById("payg-consumption").textContent="$0.00";
    });

    function loadWallet(uid){
      fetch("/api/pay/wallet?user_id="+uid)
      .then(function(r){return r.json()})
      .then(function(d){
        if(d.success){
          document.getElementById("payg-balance").textContent="$"+d.balance.toFixed(2);
          var bt=document.getElementById("payg-balance-top");if(bt)bt.textContent="$"+d.balance.toFixed(2);
          document.getElementById("payg-recharge").textContent="$"+d.total_recharge.toFixed(2);
          document.getElementById("payg-consumption").textContent="$"+d.total_consumption.toFixed(2);
          renderOrders(d.orders);
        }
      }).catch(function(){});
    }

    var _allOrders=[];
    function renderOrders(orders){
      _allOrders=orders||[];
      _filterOrders();
    }
    function _filterOrders(){
      var el=document.getElementById("payg-orders");
      var sf=document.getElementById("payg-search");
      var ff=document.getElementById("payg-filter");
      var lf=document.getElementById("payg-limit");
      var kw=sf?sf.value.trim().toLowerCase():"";
      var st=ff?ff.value:"";
      var lim=lf?parseInt(lf.value):20;
      var filtered=[];
      for(var i=0;i<_allOrders.length;i++){
        var o=_allOrders[i];
        if(st&&o.status!==st)continue;
        if(kw&&(o.sn||"").toLowerCase().indexOf(kw)===-1)continue;
        filtered.push(o);
      }
      var show=filtered.slice(0,lim);
      if(!show.length){el.innerHTML='<div style="text-align:center;padding:40px 0;color:#9ca3af;font-size:13px;">暂无匹配记录</div>';return;}
      var html='<div style="overflow-x:auto;-webkit-overflow-scrolling:touch;"><table style="width:100%;border-collapse:collapse;font-size:13px;white-space:nowrap;">'
        +'<thead><tr style="text-align:left;color:#6b7280;">'
        +'<th style="padding:10px 12px;border-bottom:1px solid #e5e7eb;">时间</th>'
        +'<th style="padding:10px 12px;border-bottom:1px solid #e5e7eb;">金额</th>'
        +'<th style="padding:10px 12px;border-bottom:1px solid #e5e7eb;">状态</th>'
        +'<th style="padding:10px 12px;border-bottom:1px solid #e5e7eb;">订单号</th>'
        +'</tr></thead><tbody>';
      for(var i=0;i<show.length;i++){
        var o=show[i];
        var t=o.created_at?o.created_at.substring(0,19).replace("T"," "):"";
        var statusText=o.status==="PAID"?"已支付":"待支付";
        var statusColor=o.status==="PAID"?"#10b981":"#f59e0b";
        html+='<tr style="border-bottom:1px solid #f3f4f6;">'
          +'<td style="padding:10px 12px;color:#6b7280;">'+t+'</td>'
          +'<td style="padding:10px 12px;font-weight:600;">\u00a5'+o.amount_yuan.toFixed(2)+'</td>'
          +'<td style="padding:10px 12px;"><span style="color:'+statusColor+';font-weight:500;">'+statusText+'</span></td>'
          +'<td style="padding:10px 12px;color:#9ca3af;font-size:12px;">'+o.sn+'</td>'
          +'</tr>';
      }
      html+='</tbody></table></div>';
      if(filtered.length>lim)html+='<div style="text-align:center;padding:8px;color:#9ca3af;font-size:12px;">显示前'+lim+'条，共'+filtered.length+'条</div>';
      el.innerHTML=html;
    }

    /* 金额按钮选中 */
    var amtBtns=document.querySelectorAll(".payg-amt");
    var customInput=document.getElementById("payg-custom");
    for(var j=0;j<amtBtns.length;j++){
      amtBtns[j].onclick=function(){
        for(var k=0;k<amtBtns.length;k++){
          amtBtns[k].style.borderColor="#e5e7eb";
          amtBtns[k].style.background="white";
          amtBtns[k].style.color="#374151";
          amtBtns[k].removeAttribute("data-selected");
        }
        this.style.borderColor="#3b82f6";
        this.style.background="#eff6ff";
        this.style.color="#3b82f6";
        this.setAttribute("data-selected","1");
        if(customInput)customInput.value="";
      };
    }
    if(customInput){
      customInput.onfocus=function(){
        for(var k=0;k<amtBtns.length;k++){
          amtBtns[k].style.borderColor="#e5e7eb";
          amtBtns[k].style.background="white";
          amtBtns[k].style.color="#374151";
          amtBtns[k].removeAttribute("data-selected");
        }
      };
    }
    /* Tab切换 */
    var tabs=document.querySelectorAll(".payg-tab");
    for(var j=0;j<tabs.length;j++){
      tabs[j].onclick=function(){
        for(var k=0;k<tabs.length;k++){
          tabs[k].style.color="#6b7280";
          tabs[k].style.borderBottom="none";
          tabs[k].style.fontWeight="500";
        }
        this.style.color="#3b82f6";
        this.style.borderBottom="2px solid #3b82f6";
        this.style.fontWeight="600";
      };
    }
    /* 搜索/筛选/条数 事件绑定 */
    var _sf=document.getElementById("payg-search");
    var _ff=document.getElementById("payg-filter");
    var _lf=document.getElementById("payg-limit");
    if(_sf)_sf.oninput=function(){_filterOrders();};
    if(_ff)_ff.onchange=function(){_filterOrders();};
    if(_lf)_lf.onchange=function(){_filterOrders();};
    /* 立即支付 -> 跳转二维码页面 */
    var sub=document.getElementById("payg-submit");
    if(sub){sub.onclick=function(){
      var amt="";
      var sel=document.querySelector(".payg-amt[data-selected]");
      if(sel)amt=sel.getAttribute("data-val");
      var cv=document.getElementById("payg-custom");
      if(cv&&cv.value)amt=cv.value;
      if(!amt){alert("请选择或输入充值金额");return;}
      var payUrl="/custom-js/pay.html?amount="+encodeURIComponent(amt);
      if(paygUserId)payUrl+="&user_id="+paygUserId;
      window.open(payUrl,"_blank");
    }}
    /* 监听支付成功消息，自动刷新余额 */
    window.addEventListener("message",function(e){
      if(e.data==="payg_paid"&&paygUserId){loadWallet(paygUserId);}
    });
  }
};

ref.parentNode.insertBefore(a,ref.nextSibling);

/* 如果当前URL含有#payg-wallet，自动打开钱包页面 */
if(location.hash==="#payg-wallet"){a.click();}

}).observe(document.body,{childList:true,subtree:true});
