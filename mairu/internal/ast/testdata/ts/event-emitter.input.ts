type Listener = (...args: any[]) => void;

export class EventEmitter {
  private listeners: Map<string, Set<Listener>> = new Map();

  /** Registers a listener for the given event name. */
  on(event: string, listener: Listener): void {
    if (!this.listeners.has(event)) {
      this.listeners.set(event, new Set());
    }
    this.listeners.get(event)!.add(listener);
  }

  /** Removes a previously registered listener. */
  off(event: string, listener: Listener): void {
    const set = this.listeners.get(event);
    if (set) {
      set.delete(listener);
    }
  }

  /** Fires all listeners registered for the given event. */
  emit(event: string, ...args: any[]): void {
    const set = this.listeners.get(event);
    if (!set) {
      return;
    }
    for (const fn of set) {
      fn(...args);
    }
  }

  /** Removes all listeners for a given event, or all events if none specified. */
  clear(event?: string): void {
    if (event) {
      this.listeners.delete(event);
    } else {
      this.listeners.clear();
    }
  }
}
