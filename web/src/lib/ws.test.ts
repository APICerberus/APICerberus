import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { ReconnectingWebSocketClient } from "./ws";

// Minimal WebSocket mock
function createMockWebSocket() {
  const listeners: Record<string, EventListener[]> = {};
  const ws = {
    readyState: 0, // CONNECTING
    send: vi.fn(),
    close: vi.fn(() => {
      ws.readyState = 3; // CLOSED
      listeners.close?.forEach((fn) => fn(new Event("close")));
    }),
    addEventListener: vi.fn((event: string, fn: EventListener) => {
      listeners[event] = listeners[event] ?? [];
      listeners[event].push(fn);
    }),
    removeEventListener: vi.fn((event: string, fn: EventListener) => {
      listeners[event] = (listeners[event] ?? []).filter((f) => f !== fn);
    }),
    // Test helpers
    simulateOpen() {
      ws.readyState = 1; // OPEN
      listeners.open?.forEach((fn) => fn(new Event("open")));
    },
    simulateMessage(data: string) {
      listeners.message?.forEach((fn) =>
        fn(new MessageEvent("message", { data })),
      );
    },
    simulateError() {
      listeners.error?.forEach((fn) => fn(new Event("error")));
    },
    simulateClose() {
      ws.readyState = 3;
      listeners.close?.forEach((fn) => fn(new CloseEvent("close")));
    },
  };
  return ws;
}

describe("ReconnectingWebSocketClient", () => {
  let client: ReconnectingWebSocketClient;
  let mockWs: ReturnType<typeof createMockWebSocket>;

  beforeEach(() => {
    vi.useFakeTimers();
    client = new ReconnectingWebSocketClient({ url: "ws://localhost:8080/ws" });

    // Intercept WebSocket constructor
    mockWs = createMockWebSocket();
    const WSConstructor = vi.fn(() => {
      mockWs.readyState = 0;
      return mockWs;
    }) as unknown as typeof WebSocket;
    // Preserve static constants
    WSConstructor.CONNECTING = 0;
    WSConstructor.OPEN = 1;
    WSConstructor.CLOSING = 2;
    WSConstructor.CLOSED = 3;
    vi.stubGlobal("WebSocket", WSConstructor);
  });

  afterEach(() => {
    client.disconnect();
    vi.useRealTimers();
    vi.restoreAllMocks();
  });

  it("starts in idle status", () => {
    expect(client.getStatus()).toBe("idle");
  });

  it("transitions to connecting then open", () => {
    const statuses: string[] = [];
    client.onStatusChange((s) => statuses.push(s));

    client.connect();
    expect(statuses).toContain("connecting");

    mockWs.simulateOpen();
    expect(client.getStatus()).toBe("open");
    expect(statuses).toContain("open");
  });

  it("receives and parses JSON messages", () => {
    const received: unknown[] = [];
    client.subscribe((msg) => received.push(msg));

    client.connect();
    mockWs.simulateOpen();

    mockWs.simulateMessage(JSON.stringify({ type: "test", data: 42 }));
    expect(received).toHaveLength(1);
    expect(received[0]).toEqual({ type: "test", data: 42 });
  });

  it("delivers raw string for non-JSON messages", () => {
    const received: unknown[] = [];
    client.subscribe((msg) => received.push(msg));

    client.connect();
    mockWs.simulateOpen();

    mockWs.simulateMessage("plain text");
    expect(received).toHaveLength(1);
    expect(received[0]).toBe("plain text");
  });

  it("sends string data when open", () => {
    client.connect();
    mockWs.readyState = 0; // Start as CONNECTING
    mockWs.simulateOpen(); // Sets readyState to 1 (OPEN)

    const sent = client.send("hello");
    expect(sent).toBe(true);
    expect(mockWs.send).toHaveBeenCalledWith("hello");
  });

  it("sends object data as JSON", () => {
    client.connect();
    mockWs.readyState = 0;
    mockWs.simulateOpen();

    client.send({ action: "ping" });
    expect(mockWs.send).toHaveBeenCalledWith(JSON.stringify({ action: "ping" }));
  });

  it("returns false when sending on closed socket", () => {
    const sent = client.send("test");
    expect(sent).toBe(false);
  });

  it("transitions to closed on disconnect", () => {
    const statuses: string[] = [];
    client.onStatusChange((s) => statuses.push(s));

    client.connect();
    mockWs.simulateOpen();
    client.disconnect();

    expect(client.getStatus()).toBe("closed");
    expect(statuses).toContain("closed");
  });

  it("notifies error listeners", () => {
    const errors: Event[] = [];
    client.onError((e) => errors.push(e));

    client.connect();
    mockWs.simulateOpen();
    mockWs.simulateError();

    expect(errors).toHaveLength(1);
  });

  it("unsubscribes message listener", () => {
    const received: unknown[] = [];
    const unsub = client.subscribe((msg) => received.push(msg));

    client.connect();
    mockWs.simulateOpen();

    mockWs.simulateMessage('{"a":1}');
    expect(received).toHaveLength(1);

    unsub();
    mockWs.simulateMessage('{"b":2}');
    expect(received).toHaveLength(1); // No new message after unsubscribe
  });

  it("does not reconnect after manual disconnect", () => {
    client.connect();
    mockWs.simulateOpen();

    client.disconnect();
    mockWs.simulateClose(); // Simulate underlying close after disconnect

    vi.advanceTimersByTime(5000);
    expect(client.getStatus()).toBe("closed");
  });

  it("attempts reconnection with backoff after unexpected close", () => {
    client.connect();
    mockWs.simulateOpen();

    // Simulate unexpected close
    mockWs.simulateClose();
    expect(client.getStatus()).toBe("reconnecting");

    // Advance past backoff delay (default: 500ms initial)
    vi.advanceTimersByTime(1000);

    // A new connect call should have been made
    expect(WebSocket).toHaveBeenCalledTimes(2);
  });

  it("stops reconnecting after max attempts", () => {
    const limitedClient = new ReconnectingWebSocketClient({
      url: "ws://localhost:8080/ws",
      maxReconnectAttempts: 2,
      reconnectInitialDelayMs: 10,
      reconnectMaxDelayMs: 100,
    });

    const statuses: string[] = [];
    limitedClient.onStatusChange((s) => statuses.push(s));

    limitedClient.connect();
    mockWs.simulateOpen();

    // Simulate 3 close events (exceeds maxReconnectAttempts=2)
    for (let i = 0; i < 3; i++) {
      mockWs.simulateClose();
      vi.advanceTimersByTime(200);
    }

    expect(limitedClient.getStatus()).toBe("closed");
    limitedClient.disconnect();
  });
});
