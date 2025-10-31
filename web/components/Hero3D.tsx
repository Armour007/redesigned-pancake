"use client"
import { Canvas, useFrame } from '@react-three/fiber'
import { useRef } from 'react'
import * as THREE from 'three'

function Knot() {
  const ref = useRef<THREE.Mesh>(null!)
  useFrame((_state, dt) => {
    if (!ref.current) return
    ref.current.rotation.x += dt * 0.25
    ref.current.rotation.y += dt * 0.35
  })
  return (
    <mesh ref={ref} position={[0, 0, 0]}>
      <torusKnotGeometry args={[1.2, 0.35, 128, 32]} />
      <meshStandardMaterial color="#7B61FF" metalness={0.6} roughness={0.3} emissive="#2c1b7f" emissiveIntensity={0.6} />
    </mesh>
  )
}

export default function Hero3D() {
  return (
    <div className="relative w-full h-[360px] md:h-[440px] lg:h-[520px]">
      <div className="absolute inset-0 -z-10 opacity-40" style={{ background: 'radial-gradient(900px circle at 50% 20%, rgba(0,224,255,0.15), transparent 40%), radial-gradient(700px circle at 80% 30%, rgba(123,97,255,0.15), transparent 40%)' }} />
      <Canvas camera={{ position: [0, 0, 4.2], fov: 50 }} dpr={[1, 2]}>
        <color attach="background" args={[0, 0, 0, 0]} />
        <ambientLight intensity={0.6} />
        <directionalLight position={[4, 6, 5]} intensity={1.2} color={new THREE.Color('#007AFF')} />
        <directionalLight position={[-4, -2, -5]} intensity={0.5} color={new THREE.Color('#7B61FF')} />
        <Knot />
      </Canvas>
    </div>
  )
}
