"use client"
import { motion } from 'framer-motion'

export default function Loader({ label = 'Loading...' }: { label?: string }) {
  return (
    <div className="inline-flex items-center gap-3">
      <div className="relative inline-flex items-center gap-2">
        {[0, 1, 2].map((i) => (
          <motion.span
            key={i}
            className="w-2.5 h-2.5 rounded-full bg-plasma"
            animate={{ y: [0, -6, 0], opacity: [0.6, 1, 0.6] }}
            transition={{ duration: 0.9, repeat: Infinity, delay: i * 0.15, ease: 'easeInOut' }}
          />
        ))}
      </div>
      <span className="text-sm text-white/70">{label}</span>
    </div>
  )
}
