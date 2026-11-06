package vm

import (
	"testing"
)

func TestSpawnAndChannels(t *testing.T) {
	tests := []vmTestCase{
		{
			input: `
			let ch = chan(1);
			ch <- 100;
			<-ch;
			`,
			expected: 100,
		},
		{
			input: `
			let ch = chan();
			spawn fn() {
				ch <- 42;
			}();
			<-ch;
			`,
			expected: 42,
		},
		{
			input: `
			let ch = chan(2);
			ch <- 10;
			ch <- 20;
			let a = <-ch;
			let b = <-ch;
			a + b;
			`,
			expected: 30,
		},
		{
			input: `
			let ch = chan(5);
			ch.capacity;
			`,
			expected: 5,
		},
		{
			input: `
			let ch = chan(1);
			ch <- "ping";
			close(ch);
			<-ch;
			`,
			expected: "ping",
		},
		{
			input: `
			let ch = chan(2);
			let s1 = ch.try_send("first");
			let s2 = ch.try_send("second");
			let s3 = ch.try_send("third");
			let r1 = ch.try_recv();
			let r2 = ch.recv(50);
			let r3 = ch.try_recv();
			s1 && s2 && (!s3) && (r1 == "first") && (r2 == "second") && (r3 == null);
			`,
			expected: true,
		},
		{
			input: `
			let ch = chan(1);
			ch.send("buffered", 100);
			let val = ch.recv();
			ch.close();
			val;
			`,
			expected: "buffered",
		},
	}
	runVmTests(t, tests)
}

func TestMutexAndSynchronization(t *testing.T) {
	tests := []vmTestCase{
		{
			input: `
			let m = mutex();
			m.lock();
			m.unlock();
			let ok = m.try_lock();
			m.unlock();
			ok;
			`,
			expected: true,
		},
		{
			input: `
			let wg = wait_group();
			let ch = chan(1);
			wg.add(1);
			spawn fn() {
				ch <- 77;
				wg.done();
			}();
			wg.wait();
			<-ch;
			`,
			expected: 77,
		},
	}
	runVmTests(t, tests)
}

func TestTcpAndUdpSockets(t *testing.T) {
	tests := []vmTestCase{
		{
			input: `
			let srv = tcp_listen("127.0.0.1:19081");
			spawn fn() {
				let sConn = srv.accept();
				let line = sConn.read_line();
				sConn.write(line + "_reply\n");
				sConn.close();
			}();

			let client = tcp_connect("127.0.0.1:19081");
			client.write("hello_tcp\n");
			let resp = client.read_line();
			client.close();
			srv.close();
			resp;
			`,
			expected: "hello_tcp_reply",
		},
		{
			input: `
			let srv = udp_listen("127.0.0.1:19082");
			let client = udp_connect("127.0.0.1:19082");
			client.write("ping_udp");

			let packet = srv.recv_from();
			srv.close();
			client.close();
			packet["data"];
			`,
			expected: "ping_udp",
		},
	}
	runVmTests(t, tests)
}

func TestHttpClientAndServer(t *testing.T) {
	tests := []vmTestCase{
		{
			input: `
			let srv = http_server();
			srv.get("/api/greet", fn(req, res) {
				res.send("hello from server");
			});
			srv.post("/api/echo", fn(req, res) {
				res.json(req.json());
			});
			srv.get("/api/custom_status", fn(req, res) {
				res.status(201).send("created");
			});
			srv.listen("127.0.0.1:19083");

			// Test GET
			let res1 = http_get("http://127.0.0.1:19083/api/greet");
			let b1 = res1.body;

			// Test POST
			let res2 = http_post("http://127.0.0.1:19083/api/echo", "{\"title\": \"Bramand\"}");
			let parsed = res2.json();
			let title = parsed["title"];

			// Test Status
			let res3 = http_get("http://127.0.0.1:19083/api/custom_status");
			let code = res3.status;

			srv.close();
			b1 == "hello from server" && title == "Bramand" && code == 201;
			`,
			expected: true,
		},
	}
	runVmTests(t, tests)
}

func TestWebSocketClientAndServer(t *testing.T) {
	tests := []vmTestCase{
		{
			input: `
			let srv = ws_server("127.0.0.1:19084", "/chat");
			spawn fn() {
				let sConn = srv.accept();
				let msg = sConn.recv();
				sConn.send("ack:" + msg);
				sConn.close();
			}();

			let client = ws_connect("ws://127.0.0.1:19084/chat");
			client.send("hello_bramand_ws");
			let reply = client.recv();
			client.close();
			srv.close();
			reply;
			`,
			expected: "ack:hello_bramand_ws",
		},
	}
	runVmTests(t, tests)
}
