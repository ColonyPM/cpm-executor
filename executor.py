import argparse
import sys

from crypto import Crypto
from pycolonies import Colonies, ColoniesConnectionError


def run_executor():
    parser = argparse.ArgumentParser(description="Colonies Python Executor")
    parser.add_argument("--host", required=True)
    parser.add_argument("--port", type=int, required=True)
    parser.add_argument("--tls", action="store_true")
    parser.add_argument("--prvkey", required=True)
    parser.add_argument("--executor-name", required=True)
    args = parser.parse_args()

    client = Colonies(args.host, args.port, tls=args.tls)
    colony_prvkey = args.prvkey

    crypto = Crypto()
    executor_prvkey = crypto.prvkey()
    executorid = crypto.id(executor_prvkey)

    executor = {
        "executorname": args.executor_name,
        "executorid": executorid,
        "colonyname": "dev",
        "executortype": "python-executor",
    }

    print(f"Attempting to register executor: {executor['executorname']}")
    try:
        client.add_executor(executor, colony_prvkey)
        client.approve_executor("dev", executor["executorname"], colony_prvkey)
        print(f"✅ {executor['executorname']} registered and approved.")
    except Exception as e:
        print(f"ℹ️ Registration skipped (likely already exists): {e}")

    print("🚀 Executor started. Press Ctrl+C to stop.")

    try:
        while True:
            process = None
            try:
                # 10s timeout to check for tasks
                process = client.assign("dev", 10, executor_prvkey)
            except ColoniesConnectionError:
                print("⚠️ No process assigned, retrying...")
            except Exception:
                # Silently catch 'no process found' timeouts
                pass

            if process:
                print(f"🛠️ Processing task {process.processid}...")
                result = process.spec.args[0]
                client.close(process.processid, [result], executor_prvkey)
                print(f"✅ Closed task. Result: {result}")

    except KeyboardInterrupt:
        print("\n👋 Shutdown requested. Exiting...")
        sys.exit(0)


if __name__ == "__main__":
    run_executor()
