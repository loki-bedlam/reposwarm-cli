import chalk from 'chalk'
import { Command } from 'commander'

function getRootOpts(cmd: Command): Record<string, unknown> {
  let root = cmd
  while (root.parent) root = root.parent
  return root.opts()
}

export function isJson(cmd: Command): boolean {
  return getRootOpts(cmd).json === true
}

export function getGlobalOpts(cmd: Command) {
  const opts = getRootOpts(cmd)
  return {
    json: opts.json === true,
    region: (opts.region as string) || 'us-east-1',
    profile: opts.profile as string | undefined,
    table: (opts.table as string) || 'reposwarm-cache',
  }
}

export function output(cmd: Command, data: unknown, humanFn?: (data: any) => void) {
  if (isJson(cmd)) {
    console.log(JSON.stringify(data, null, 2))
  } else if (humanFn) {
    humanFn(data)
  } else {
    console.log(data)
  }
}

export function error(cmd: Command, message: string, details?: unknown) {
  if (isJson(cmd)) {
    console.error(JSON.stringify({ error: message, details }, null, 2))
  } else {
    console.error(chalk.red(`Error: ${message}`))
    if (details) console.error(details)
  }
  process.exit(1)
}

export function success(message: string) {
  console.log(chalk.green(`✓ ${message}`))
}

export function info(message: string) {
  console.log(chalk.blue(`ℹ ${message}`))
}

export function warn(message: string) {
  console.log(chalk.yellow(`⚠ ${message}`))
}
