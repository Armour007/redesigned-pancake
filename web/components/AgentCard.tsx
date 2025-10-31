"use client"
import { motion } from 'framer-motion'

export default function AgentCard({ name, status }: { name: string; status: string }) {
  return (
    <motion.div whileHover={{ y: -4 }} className="rounded-2xl border border-white/10 p-5 bg-white/5">
      <div className="flex items-center justify-between">
        <div>
          <div className="text-lg font-semibold">{name}</div>
          <div className="text-xs text-white/60">{status}</div>
        </div>
        <div className="w-10 h-10 rounded-xl bg-auraGradient breathing-glow" />
      </div>
    </motion.div>
  )
}
