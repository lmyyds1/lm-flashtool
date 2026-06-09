import socket
import threading
import time

def handle_client(conn, addr):
    """处理客户端连接"""
    print(f"客户端连接: {addr}")
    try:
        while True:
            data = conn.recv(1024)
            if not data:
                break
            print(f"收到数据: {data.decode()}")
            conn.sendall(b"OK")
    finally:
        conn.close()
        print(f"客户端断开: {addr}")

def start_server(port=5037):
    """启动 TCP 服务器"""
    server = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    server.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
    
    try:
        server.bind(('0.0.0.0', port))
        server.listen(5)
        print(f"服务器已启动，监听端口 {port}")
        print("按 Ctrl+C 停止服务器")
        
        while True:
            conn, addr = server.accept()
            client_thread = threading.Thread(target=handle_client, args=(conn, addr))
            client_thread.daemon = True
            client_thread.start()
            
    except KeyboardInterrupt:
        print("\n服务器正在关闭...")
    except Exception as e:
        print(f"启动失败: {e}")
    finally:
        server.close()
        print("服务器已关闭")

if __name__ == "__main__":
    start_server(5037)