export default {
  LogLevel: process.env.LOG_LEVEL ?? 'info',
  DryRun: process.env.DRY_RUN ? !(process.env.DRY_RUN === 'false' || process.env.DRY_RUN === '0') : false,
}