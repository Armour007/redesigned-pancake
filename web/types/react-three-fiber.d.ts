// Lightweight fallback typings for R3F JSX elements to keep TS happy in Next.js
// If you upgrade @react-three/fiber and three types, consider removing this file.
import '@react-three/fiber'

declare global {
  namespace JSX {
    interface IntrinsicElements {
      mesh: any
      torusKnotGeometry: any
      meshStandardMaterial: any
      ambientLight: any
      directionalLight: any
      color: any
    }
  }
}
export {}
