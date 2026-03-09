import type { WSEvent } from "@/shared/types";
import { API_BASE_URL, WS_URL } from "@/shared/lib/runtime";

type ConnectionState =
  | "connecting"
  | "connected"
  | "disconnected"
  | "reconnecting";

type ClientEventMap = {
  connected: void;
  disconnected: { code?: number; reason?: string };
  message: WSEvent;
  error: Event | Error;
  reconnecting: { attempt: number; delay: number };
  state: ConnectionState;
};

type Handler<T> = (payload: T) => void;

export class WSClient {
  private ws: WebSocket | null = null;
  private reconnectAttempts = 0;
  private readonly maxReconnectAttempts = 10;
  private readonly reconnectDelay = 1000;
  private pingInterval: ReturnType<typeof setInterval> | null = null;
  private reconnectTimeout: ReturnType<typeof setTimeout> | null = null;
  private messageQueue: WSEvent[] = [];
  private listeners = new Map<keyof ClientEventMap, Set<Handler<any>>>();
  private manualDisconnect = false;
  private state: ConnectionState = "disconnected";

  constructor(private readonly getToken: () => string | null) {}

  connect(): void {
    if (this.ws && (this.ws.readyState === WebSocket.OPEN || this.ws.readyState === WebSocket.CONNECTING)) {
      return;
    }

    const token = this.getToken();
    if (!token) {
      this.setState("disconnected");
      return;
    }

    this.manualDisconnect = false;
    this.setState(this.reconnectAttempts > 0 ? "reconnecting" : "connecting");
    this.ws = new WebSocket(this.buildWebSocketURL(token));

    this.ws.onopen = () => this.onOpen();
    this.ws.onmessage = (event) => this.onMessage(event.data);
    this.ws.onclose = (event) => this.onClose(event.code, event.reason);
    this.ws.onerror = (event) => this.onError(event);
  }

  disconnect(): void {
    this.manualDisconnect = true;
    this.clearReconnectTimeout();
    this.stopPing();

    if (this.ws) {
      this.ws.close(1000, "client disconnect");
      this.ws = null;
    }

    this.setState("disconnected");
  }

  send(event: WSEvent): void {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(event));
      return;
    }

    this.messageQueue.push(event);
    if (!this.manualDisconnect) {
      this.connect();
    }
  }

  on<TEvent extends keyof ClientEventMap>(
    eventType: TEvent,
    handler: Handler<ClientEventMap[TEvent]>
  ): () => void {
    const handlers =
      this.listeners.get(eventType) ?? new Set<Handler<ClientEventMap[TEvent]>>();
    handlers.add(handler as Handler<any>);
    this.listeners.set(eventType, handlers);

    return () => {
      handlers.delete(handler as Handler<any>);
      if (handlers.size === 0) {
        this.listeners.delete(eventType);
      }
    };
  }

  getState(): ConnectionState {
    return this.state;
  }

  private onOpen(): void {
    this.reconnectAttempts = 0;
    this.setState("connected");
    this.emit("connected", undefined);
    this.flushQueue();
    this.startPing();
  }

  private onMessage(data: string): void {
    try {
      const parsed = JSON.parse(data) as WSEvent;
      this.emit("message", parsed);
    } catch (error) {
      this.emit("error", error instanceof Error ? error : new Error("Invalid websocket payload"));
    }
  }

  private onClose(code: number, reason?: string): void {
    this.stopPing();
    this.ws = null;
    this.emit("disconnected", { code, reason });
    this.setState("disconnected");

    if (!this.manualDisconnect) {
      this.scheduleReconnect();
    }
  }

  private onError(error: Event): void {
    this.emit("error", error);
  }

  private scheduleReconnect(): void {
    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      return;
    }

    this.reconnectAttempts += 1;
    const delay = Math.min(
      this.reconnectDelay * 2 ** (this.reconnectAttempts - 1),
      30000
    );

    this.emit("reconnecting", { attempt: this.reconnectAttempts, delay });
    this.setState("reconnecting");
    this.clearReconnectTimeout();

    this.reconnectTimeout = window.setTimeout(() => {
      this.connect();
    }, delay);
  }

  private startPing(): void {
    this.stopPing();

    // Browsers cannot send websocket protocol ping frames directly, so this interval
    // acts as a connection watchdog and triggers reconnects if the socket stops being open.
    this.pingInterval = window.setInterval(() => {
      if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
        this.stopPing();
        if (!this.manualDisconnect) {
          this.scheduleReconnect();
        }
      }
    }, 30000);
  }

  private stopPing(): void {
    if (this.pingInterval) {
      window.clearInterval(this.pingInterval);
      this.pingInterval = null;
    }
  }

  private flushQueue(): void {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
      return;
    }

    for (const event of this.messageQueue) {
      this.ws.send(JSON.stringify(event));
    }

    this.messageQueue = [];
  }

  private buildWebSocketURL(token: string) {
    const configuredURL = WS_URL;
    if (configuredURL) {
      const separator = configuredURL.includes("?") ? "&" : "?";
      return `${configuredURL}${separator}token=${encodeURIComponent(token)}`;
    }

    const wsBase = API_BASE_URL.replace(/^http/, "ws").replace(/\/api\/v1\/?$/, "");
    return `${wsBase}/api/v1/ws?token=${encodeURIComponent(token)}`;
  }

  private setState(nextState: ConnectionState) {
    this.state = nextState;
    this.emit("state", nextState);
  }

  private emit<TEvent extends keyof ClientEventMap>(
    eventType: TEvent,
    payload: ClientEventMap[TEvent]
  ) {
    const handlers = this.listeners.get(eventType);
    if (!handlers) {
      return;
    }

    handlers.forEach((handler) => {
      handler(payload);
    });
  }

  private clearReconnectTimeout() {
    if (this.reconnectTimeout) {
      window.clearTimeout(this.reconnectTimeout);
      this.reconnectTimeout = null;
    }
  }
}

export type { ConnectionState };
