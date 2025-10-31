"use client"
import { motion } from 'framer-motion'

// AURA UI REBUILD
export default function Hero() {
  return (
    <div className="relative isolate">
      <div className="absolute inset-0 -z-10 opacity-60" style={{ background: 'radial-gradient(800px circle at 50% 20%, rgba(0,224,255,0.15), transparent 40%), radial-gradient(600px circle at 80% 30%, rgba(123,97,255,0.15), transparent 40%)' }} />
      <div className="container mx-auto max-w-6xl px-6 pt-20 pb-10 text-center">
        <motion.h1 initial={{ opacity: 0, y: 10 }} animate={{ opacity: 1, y: 0 }} transition={{ duration: 0.7 }} className="text-4xl md:text-6xl font-semibold tracking-tight">
          Trusted Intelligence for Autonomous Systems
        </motion.h1>
        <motion.p initial={{ opacity: 0 }} animate={{ opacity: 1 }} transition={{ delay: 0.3, duration: 0.7 }} className="mt-4 text-white/80 max-w-2xl mx-auto">
          Build, connect, and deploy AI agents that earn trust.
        </motion.p>
        <motion.div initial={{ opacity: 0 }} animate={{ opacity: 1 }} transition={{ delay: 0.5 }} className="mt-10 flex items-center justify-center gap-4">
          <a className="px-6 py-3 rounded-2xl bg-plasma text-black font-medium" href="/dashboard">Launch Workspace</a>
          <a className="px-6 py-3 rounded-2xl border border-white/20" href="/docs/BRAND_GUIDE.md">Read Docs</a>
        </motion.div>
      </div>
    </div>
  )
}
