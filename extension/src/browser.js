// Resolve the namespace at access time, not import time: tests install the mock after the modules load.
export const browser = new Proxy({}, {
  get(_target, prop) {
    return (globalThis.browser ?? globalThis.chrome)[prop]
  },
})
