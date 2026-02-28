import { CodeCommitClient, ListRepositoriesCommand, GetRepositoryCommand } from '@aws-sdk/client-codecommit'
import { DynamoDBClient } from '@aws-sdk/client-dynamodb'
import { DynamoDBDocumentClient, ScanCommand, PutCommand, DeleteCommand, GetCommand } from '@aws-sdk/lib-dynamodb'

export interface AwsConfig {
  region: string
  profile?: string
  table: string
}

export function createCodeCommitClient(config: AwsConfig) {
  return new CodeCommitClient({ region: config.region })
}

export function createDynamoClient(config: AwsConfig) {
  const raw = new DynamoDBClient({ region: config.region })
  return DynamoDBDocumentClient.from(raw)
}

export interface RepoInfo {
  name: string
  url: string
  source: 'CodeCommit' | 'GitHub'
  lastModified?: string
  description?: string
  enabled: boolean
  status: string
}

export async function listCodeCommitRepos(config: AwsConfig): Promise<RepoInfo[]> {
  const client = createCodeCommitClient(config)
  const repos: RepoInfo[] = []
  let nextToken: string | undefined

  do {
    const response = await client.send(new ListRepositoriesCommand({
      nextToken,
      sortBy: 'repositoryName',
      order: 'ascending'
    }))

    for (const repo of response.repositories || []) {
      if (!repo.repositoryName) continue
      try {
        const detail = await client.send(new GetRepositoryCommand({
          repositoryName: repo.repositoryName
        }))
        const meta = detail.repositoryMetadata
        repos.push({
          name: repo.repositoryName,
          url: meta?.cloneUrlHttp || `codecommit://${repo.repositoryName}`,
          source: 'CodeCommit',
          lastModified: meta?.lastModifiedDate?.toISOString(),
          description: meta?.repositoryDescription,
          enabled: true,
          status: 'active'
        })
      } catch {
        repos.push({
          name: repo.repositoryName,
          url: `codecommit://${repo.repositoryName}`,
          source: 'CodeCommit',
          enabled: true,
          status: 'active'
        })
      }
    }

    nextToken = response.nextToken
  } while (nextToken)

  return repos
}

export async function listTrackedRepos(config: AwsConfig): Promise<RepoInfo[]> {
  const client = createDynamoClient(config)
  const response = await client.send(new ScanCommand({
    TableName: config.table,
    FilterExpression: '#sk = :marker',
    ExpressionAttributeNames: { '#sk': 'analysis_timestamp' },
    ExpressionAttributeValues: { ':marker': 0 }
  }))

  return (response.Items || []).map(item => ({
    name: item.repository_name || '',
    url: item.url || '',
    source: (item.source || 'CodeCommit') as 'CodeCommit' | 'GitHub',
    lastModified: item.lastAnalyzed || item.lastCommit,
    enabled: item.enabled !== false,
    status: item.status || 'active'
  }))
}

export async function addTrackedRepo(config: AwsConfig, repo: RepoInfo): Promise<void> {
  const client = createDynamoClient(config)
  await client.send(new PutCommand({
    TableName: config.table,
    Item: {
      repository_name: repo.name,
      analysis_timestamp: 0,
      url: repo.url,
      source: repo.source,
      enabled: repo.enabled,
      status: repo.status,
      description: repo.description,
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString()
    }
  }))
}

export async function removeTrackedRepo(config: AwsConfig, name: string): Promise<void> {
  const client = createDynamoClient(config)
  await client.send(new DeleteCommand({
    TableName: config.table,
    Key: { repository_name: name, analysis_timestamp: 0 }
  }))
}

export async function getTrackedRepo(config: AwsConfig, name: string): Promise<RepoInfo | null> {
  const client = createDynamoClient(config)
  const response = await client.send(new GetCommand({
    TableName: config.table,
    Key: { repository_name: name, analysis_timestamp: 0 }
  }))
  if (!response.Item) return null
  return {
    name: response.Item.repository_name,
    url: response.Item.url || '',
    source: response.Item.source || 'CodeCommit',
    lastModified: response.Item.lastAnalyzed,
    enabled: response.Item.enabled !== false,
    status: response.Item.status || 'active'
  }
}

export async function updateTrackedRepo(config: AwsConfig, name: string, updates: Partial<RepoInfo>): Promise<void> {
  const existing = await getTrackedRepo(config, name)
  if (!existing) throw new Error(`Repository '${name}' not found`)

  const client = createDynamoClient(config)
  await client.send(new PutCommand({
    TableName: config.table,
    Item: {
      repository_name: name,
      analysis_timestamp: 0,
      url: updates.url ?? existing.url,
      source: updates.source ?? existing.source,
      enabled: updates.enabled ?? existing.enabled,
      status: updates.status ?? existing.status,
      updatedAt: new Date().toISOString()
    }
  }))
}
