#!/usr/bin/env python3
"""收钱吧支付服务 - Flask API"""
import hashlib, json, time, uuid, io, base64, os
import urllib.request, urllib.error
from flask import Flask, request, jsonify, Response
import qrcode
import redis

app = Flask(__name__)

# 收钱吧配置
TERMINAL_SN = "100108880053500132"
TERMINAL_KEY = "c5962b1702039a8702c106468f9cd2e9"
API_BASE = "https://vsi-api.shouqianba.com"

# 收钱吧公钥 (用于回调验签)
SQB_PUBLIC_KEY = """-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA5+MNqcjgw4bsSWhJfw2M
+gQB7P+pEiYOfvRmA6kt7Wisp0J3JbOtsLXGnErn5ZY2D8KkSAHtMYbeddphFZQJ
zUbiaDi75GUAG9XS3MfoKAhvNkK15VcCd8hFgNYCZdwEjZrvx6Zu1B7c29S64LQP
HceS0nyXF8DwMIVRcIWKy02cexgX0UmUPE0A2sJFoV19ogAHaBIhx5FkTy+eeBJE
bU03Do97q5G9IN1O3TssvbYBAzugz+yUPww2LadaKexhJGg+5+ufoDd0+V3oFL0/
ebkJvD0uiBzdE3/ci/tANpInHAUDIHoWZCKxhn60f3/3KiR8xuj2vASgEqphxT5O
fwIDAQAB
-----END PUBLIC KEY-----"""

# 数据库配置
import psycopg2
DB_CONFIG = {
    "host": "127.0.0.1",
    "port": 5432,
    "dbname": "sub2api",
    "user": "postgres",
    "password": "xingsuancode2026@Sub2API"
}

def get_db():
    return psycopg2.connect(**DB_CONFIG)

# Redis 连接 (Sub2API billing cache)
redis_client = redis.Redis(host="172.19.0.4", port=6379, db=0, decode_responses=True)

def sqb_sign(body_str):
    """收钱吧签名: MD5(body + terminal_key)"""
    return hashlib.md5((body_str + TERMINAL_KEY).encode()).hexdigest()

def sqb_request(path, body_dict):
    """发送请求到收钱吧"""
    body_str = json.dumps(body_dict)
    sign = sqb_sign(body_str)
    req = urllib.request.Request(
        API_BASE + path,
        data=body_str.encode(),
        headers={
            "Content-Type": "application/json",
            "Authorization": TERMINAL_SN + " " + sign
        }
    )
    try:
        resp = urllib.request.urlopen(req, timeout=15)
        return json.loads(resp.read().decode())
    except urllib.error.HTTPError as e:
        return json.loads(e.read().decode())

def gen_qr_base64(data):
    """生成二维码的base64图片"""
    img = qrcode.make(data, box_size=8, border=2)
    buf = io.BytesIO()
    img.save(buf, format="PNG")
    return base64.b64encode(buf.getvalue()).decode()

def _invalidate_auth_cache(user_id):
    """清除用户所有 API Key 的认证缓存（与 Go 端 InvalidateAuthCacheByUserID 等效）
    步骤：查 api_keys 表 -> SHA256(key) -> 删 Redis L2 -> Pub/Sub 通知 Go 清 L1"""
    try:
        conn = get_db()
        cur = conn.cursor()
        cur.execute("SELECT key FROM api_keys WHERE user_id=%s AND deleted_at IS NULL", (user_id,))
        keys = [r[0] for r in cur.fetchall()]
        cur.close()
        conn.close()
        for k in keys:
            cache_key = hashlib.sha256(k.encode()).hexdigest()
            redis_client.delete(f"apikey:auth:{cache_key}")
            redis_client.publish("auth:cache:invalidate", cache_key)
        if keys:
            print(f"Auth cache invalidated for user {user_id}, {len(keys)} key(s)")
    except Exception as e:
        print(f"Auth cache invalidate error: {e}")

@app.route("/api/pay/precreate", methods=["POST"])
def precreate():
    """预下单 - 生成支付二维码"""
    data = request.get_json(force=True)
    amount_yuan = data.get("amount")
    user_id = data.get("user_id")
    payway = data.get("payway", "1")  # 1:支付宝 3:微信
    if not amount_yuan:
        return jsonify({"error": "缺少金额"}), 400

    amount_cent = str(int(float(amount_yuan) * 100))
    client_sn = "XS" + str(int(time.time()*1000)) + str(uuid.uuid4().hex[:6])

    body = {
        "terminal_sn": TERMINAL_SN,
        "client_sn": client_sn,
        "total_amount": amount_cent,
        "payway": payway,
        "subject": "星算code充值 ¥" + str(amount_yuan),
        "operator": "system",
        "reflect": json.dumps({"user_id": user_id or "", "amount_yuan": str(amount_yuan)})
    }

    result = sqb_request("/upay/v2/precreate", body)

    if result.get("result_code") == "200" and result.get("biz_response", {}).get("result_code") == "PRECREATE_SUCCESS":
        qr_code = result["biz_response"]["data"]["qr_code"]
        sn = result["biz_response"]["data"]["sn"]
        qr_img = gen_qr_base64(qr_code)

        # 不在预下单时写入数据库，只返回二维码信息
        # 订单记录将在支付成功时由 _handle_paid 创建

        return jsonify({
            "success": True,
            "qr_code": qr_code,
            "qr_img": qr_img,
            "sn": sn,
            "client_sn": client_sn
        })
    else:
        return jsonify({"success": False, "error": result}), 500

@app.route("/api/pay/query", methods=["GET"])
def query_order():
    """查询订单状态"""
    sn = request.args.get("sn")
    client_sn = request.args.get("client_sn")
    if not sn and not client_sn:
        return jsonify({"error": "缺少sn或client_sn"}), 400

    body = {"terminal_sn": TERMINAL_SN}
    if sn:
        body["sn"] = sn
    else:
        body["client_sn"] = client_sn

    result = sqb_request("/upay/v2/query", body)

    if result.get("result_code") == "200":
        biz = result.get("biz_response") or {}
        data = biz.get("data") or {}
        order_status = data.get("order_status", "UNKNOWN")

        # 如果支付成功，更新数据库并充值
        if order_status == "PAID":
            _handle_paid(data)

        return jsonify({
            "success": True,
            "order_status": order_status,
            "total_amount": data.get("total_amount"),
            "payway_name": data.get("payway_name", "")
        })
    else:
        return jsonify({"success": False, "error": result}), 500

def _handle_paid(data):
    """支付成功处理：写入订单记录 + 充值余额（仅在实际支付成功时才记录）"""
    sn = data.get("sn", "")
    client_sn = data.get("client_sn", "")
    reflect = {}
    try:
        reflect = json.loads(data.get("reflect", "{}"))
    except Exception:
        pass
    user_id = reflect.get("user_id") or None
    amount_yuan_str = reflect.get("amount_yuan", "0")
    total_amount = data.get("total_amount", "0")
    amount_yuan = round(int(total_amount) / 100, 2) if total_amount else float(amount_yuan_str)
    amount_cent = int(total_amount) if total_amount else int(float(amount_yuan_str) * 100)

    try:
        conn = get_db()
        cur = conn.cursor()
        # 检查是否已处理过该sn
        cur.execute("SELECT id FROM payg_orders WHERE sn=%s", (sn,))
        row = cur.fetchone()
        if row:
            # 已存在，跳过
            cur.close()
            conn.close()
            return
        # 插入已支付订单记录
        cur.execute("""
            INSERT INTO payg_orders (client_sn, sn, user_id, amount_yuan, amount_cent, status, created_at, paid_at)
            VALUES (%s, %s, %s, %s, %s, 'PAID', NOW(), NOW())
        """, (client_sn, sn, user_id if user_id else None, amount_yuan, amount_cent))
        # 充值余额 (1:1，人民币=美金，通过倍率控制)
        usd_amount = round(float(amount_yuan), 2)
        if user_id:
            cur.execute("UPDATE users SET balance = balance + %s WHERE id = %s", (usd_amount, user_id))
        conn.commit()
        cur.close()
        conn.close()
        # 清除 Sub2API 的 Redis 余额缓存，使新余额立即生效
        if user_id:
            try:
                redis_client.delete(f"billing:balance:{user_id}")
            except Exception as re:
                print(f"Redis cache clear error: {re}")
            # 清除 API Key 认证缓存（含余额快照），与后台充值行为一致
            _invalidate_auth_cache(user_id)
    except Exception as e:
        print(f"Handle paid error: {e}")

@app.route("/api/pay/wallet", methods=["GET"])
def wallet():
    """获取用户钱包信息：余额、累计充值、充值记录"""
    user_id = request.args.get("user_id")
    if not user_id:
        return jsonify({"error": "缺少user_id"}), 400
    try:
        conn = get_db()
        cur = conn.cursor()
        # 用户余额
        cur.execute("SELECT balance FROM users WHERE id=%s", (user_id,))
        row = cur.fetchone()
        balance = float(row[0]) if row else 0
        # 累计充值
        cur.execute("SELECT COALESCE(SUM(amount_yuan),0) FROM payg_orders WHERE user_id=%s AND status='PAID'", (user_id,))
        total_recharge = float(cur.fetchone()[0])
        # 充值记录
        cur.execute("""
            SELECT amount_yuan, status, created_at, paid_at, sn
            FROM payg_orders WHERE user_id=%s ORDER BY created_at DESC LIMIT 50
        """, (user_id,))
        orders = []
        for r in cur.fetchall():
            orders.append({
                "amount_yuan": float(r[0]),
                "status": r[1],
                "created_at": r[2].isoformat() if r[2] else "",
                "paid_at": r[3].isoformat() if r[3] else "",
                "sn": r[4] or ""
            })
        cur.close()
        conn.close()
        return jsonify({
            "success": True,
            "balance": round(balance, 2),
            "total_recharge": round(total_recharge, 2),
            "total_consumption": round(total_recharge - balance, 2) if total_recharge > balance else 0,
            "orders": orders
        })
    except Exception as e:
        return jsonify({"success": False, "error": str(e)}), 500

@app.route("/api/pay/admin/wallet", methods=["GET"])
def admin_wallet():
    """管理员视图：全平台充值统计 + 按用户汇总"""
    try:
        conn = get_db()
        cur = conn.cursor()
        # 全平台统计
        cur.execute("SELECT COUNT(*), COALESCE(SUM(CASE WHEN status='PAID' THEN 1 ELSE 0 END),0), COALESCE(SUM(CASE WHEN status='PAID' THEN amount_yuan ELSE 0 END),0), COALESCE(SUM(CASE WHEN status!='PAID' THEN 1 ELSE 0 END),0) FROM payg_orders")
        row = cur.fetchone()
        total_orders = row[0]
        paid_orders = int(row[1])
        total_recharge = float(row[2])
        pending_orders = int(row[3])
        # 按用户汇总
        cur.execute("""
            SELECT o.user_id, u.email, COUNT(*) as order_count,
                   COALESCE(SUM(CASE WHEN o.status='PAID' THEN o.amount_yuan ELSE 0 END),0) as total_paid
            FROM payg_orders o LEFT JOIN users u ON o.user_id = u.id
            GROUP BY o.user_id, u.email
            ORDER BY total_paid DESC
        """)
        users = []
        for r in cur.fetchall():
            users.append({
                "user_id": r[0],
                "email": r[1] or "unknown",
                "order_count": r[2],
                "total_recharge": float(r[3])
            })
        # 最近100条订单（含用户邮箱）
        cur.execute("""
            SELECT o.amount_yuan, o.status, o.created_at, o.paid_at, o.sn, u.email, o.user_id
            FROM payg_orders o LEFT JOIN users u ON o.user_id = u.id
            ORDER BY o.created_at DESC LIMIT 100
        """)
        orders = []
        for r in cur.fetchall():
            orders.append({
                "amount_yuan": float(r[0]),
                "status": r[1],
                "created_at": r[2].isoformat() if r[2] else "",
                "paid_at": r[3].isoformat() if r[3] else "",
                "sn": r[4] or "",
                "email": r[5] or "unknown",
                "user_id": r[6]
            })
        cur.close()
        conn.close()
        return jsonify({
            "success": True,
            "total_recharge": round(total_recharge, 2),
            "total_orders": total_orders,
            "paid_orders": paid_orders,
            "pending_orders": pending_orders,
            "users": users,
            "orders": orders
        })
    except Exception as e:
        return jsonify({"success": False, "error": str(e)}), 500

@app.route("/api/pay/callback", methods=["POST"])
def callback():
    """收钱吧支付回调"""
    data = request.get_json(force=True)
    # 处理回调
    if data.get("order_status") == "PAID":
        _handle_paid(data)
    return jsonify({"result": "success"})

if __name__ == "__main__":
    app.run(host="0.0.0.0", port=8901)
