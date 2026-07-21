export const productMode = Object.freeze({
  opMode: import.meta.env.VITE_OP_MODE === 'true',
  brand: import.meta.env.VITE_BRAND || 'sdpivot',
})
