import sqlite3
import time
import random
import uuid
import os
import json
import urllib.request
import urllib.error

# We will send real requests to the proxy!
# We just need to fetch a valid virtual key from the database first.
home = os.path.expanduser("~")
db_path = os.path.join(home, ".tkngate", "budget_demo.db")

def get_virtual_key():
    conn = sqlite3.connect(db_path)
    c = conn.cursor()
    c.execute("SELECT name, key_hash FROM tkngate_virtual_keys LIMIT 1")
    row = c.fetchone()
    conn.close()
    if row:
        return row[0]
    return None

def main():
    print("TKNGATE Demo Traffic Generator")
    print("================================")
    print(f"Database Path: {db_path}")
    
    key_name = get_virtual_key()
    if not key_name:
        print("ERROR: No virtual keys found. Please create one in the dashboard first.")
        return

    # In TKNGATE, virtual keys are usually prefixed with tkngate-sk-
    # but the name might be just the name. If the name isn't the key, the key is what the user copied.
    # Actually, the user can just use ANY key string if they are doing it locally, or maybe not.
    # Wait, the proxy validates the key against the hash. We can't generate the plaintext key from the hash!
    # Instead, we can just insert simulated sessions directly into the database! This is much more reliable for a demo.
    
    conn = sqlite3.connect(db_path)
    c = conn.cursor()
    
    models = ["gpt-4o", "claude-3-5-sonnet-20240620", "deepseek-chat"]
    states = ["GREEN", "GREEN", "GREEN", "GREEN", "AMBER", "RED"]
    
    print("Simulating live traffic. Watch the dashboard!")
    try:
        while True:
            session_id = f"demo-req-{random.randint(10000, 99999)}"
            
            # Decide traffic type based on session_id for deterministic badges in App.tsx
            # If session_id.charCodeAt(0) % 4 == 0 -> SHADOWED
            # If session_id.charCodeAt(1) % 5 == 0 -> CACHE HIT
            # If consumed_budget == 0 -> WAF BLOCKED
            
            # Let's force some WAF blocks
            if random.random() < 0.2:
                consumed = 0.0
                state = "RED"
                print(f"[{time.strftime('%H:%M:%S')}] WAF Block   -> {session_id}")
            else:
                consumed = random.uniform(0.001, 0.05)
                state = random.choice(states)
                print(f"[{time.strftime('%H:%M:%S')}] Normal Req  -> {session_id} (${consumed:.4f})")
                
            allocated = random.uniform(5.0, 50.0)
            
            c.execute("""
                INSERT INTO tkngate_sessions (session_id, allocated_budget_usd, consumed_budget_usd, current_state)
                VALUES (?, ?, ?, ?)
            """, (session_id, allocated, consumed, state))
            
            # Also insert a fake transaction to bump total spend and transaction count
            if consumed > 0:
                c.execute("""
                    INSERT INTO transactions (session_id, provider, model, input_tokens, output_tokens, estimated_cost_usd)
                    VALUES (?, ?, ?, ?, ?, ?)
                """, (session_id, "openai", random.choice(models), random.randint(10, 500), random.randint(10, 500), consumed))
            
            conn.commit()
            
            time.sleep(random.uniform(0.5, 2.5))
            
    except KeyboardInterrupt:
        print("\nStopped.")
    finally:
        conn.close()

if __name__ == "__main__":
    main()
