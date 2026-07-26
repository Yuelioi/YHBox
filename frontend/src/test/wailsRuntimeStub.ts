type EventHandler = (event: { data: unknown[] }) => void

const handlers = new Map<string, Set<EventHandler>>()

type Creator<T = unknown> = (value: unknown) => T

export const Create = {
  Any(value: unknown) {
    return value
  },

  Nullable<T>(create: Creator<T>) {
    return (value: unknown): T | null | undefined => {
      if (value == null) return value as null | undefined
      return create(value)
    }
  },

  Array<T>(create: Creator<T>) {
    return (value: unknown): T[] => {
      if (!Array.isArray(value)) return []
      return value.map((item) => create(item))
    }
  },

  Map(_keyCreate: Creator, valueCreate: Creator) {
    return (value: unknown): Record<string, unknown> => {
      if (!value || typeof value !== 'object') return {}
      return Object.fromEntries(
        Object.entries(value as Record<string, unknown>).map(([key, item]) => [
          key,
          valueCreate(item),
        ]),
      )
    }
  },

  Events: {},
}

export const Call = {
  ByID(_id: number, ..._args: unknown[]) {
    return Promise.reject(new Error('Wails runtime is not available in Vitest'))
  },
}

export const CancellablePromise = Promise

export const Events = {
  On(name: string, handler: EventHandler) {
    const bucket = handlers.get(name) ?? new Set<EventHandler>()
    bucket.add(handler)
    handlers.set(name, bucket)
    return () => {
      bucket.delete(handler)
      if (bucket.size === 0) handlers.delete(name)
    }
  },

  async Emit(name: string, data?: unknown) {
    const bucket = handlers.get(name)
    if (!bucket) return
    for (const handler of bucket) {
      handler({ data: [data] })
    }
  },
}

export const Window = {
  async IsMaximised() {
    return false
  },
  async Minimise() {},
  async ToggleMaximise() {},
  async Close() {},
}

export const Browser = {
  async OpenURL(_url: string) {},
}

export const Dialogs = {
  async OpenFile(_options: unknown) {
    return ''
  },
  async SaveFile(_options: unknown) {
    return ''
  },
}
