import { useState } from 'react'
import { createFileRoute } from '@tanstack/react-router'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Copy } from 'lucide-react'
import toast from 'react-hot-toast'
import { PageHeader } from '@/components/page-header'

const codeExamples = {
  python: `import requests

API_BASE = "https://your-gateway.example.com/v1"
API_KEY = "your-api-key-here"

headers = {
    "Authorization": f"Bearer {API_KEY}",
    "Content-Type": "application/json"
}

data = {
    "model": "gpt-4o-mini",
    "messages": [
        {"role": "user", "content": "Hello!"}
    ]
}

response = requests.post(
    f"{API_BASE}/chat/completions",
    headers=headers,
    json=data
)

print(response.json())`,

  curl: `curl -X POST "https://your-gateway.example.com/v1/chat/completions" \\
  -H "Authorization: Bearer YOUR_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "gpt-4o-mini",
    "messages": [
      {"role": "user", "content": "Hello!"}
    ]
  }'`,

  nodejs: `const axios = require('axios');

const API_BASE = 'https://your-gateway.example.com/v1';
const API_KEY = 'your-api-key-here';

async function chat() {
  const response = await axios.post(
    \`\${API_BASE}/chat/completions\`,
    {
      model: 'gpt-4o-mini',
      messages: [
        { role: 'user', content: 'Hello!' }
      ]
    },
    {
      headers: {
        'Authorization': \`Bearer \${API_KEY}\`,
        'Content-Type': 'application/json'
      }
    }
  );

  console.log(response.data);
}

chat();`,
}

const models = [
  { name: 'gpt-4o', provider: 'OpenAI', type: 'chat', contextWindow: '128K' },
  { name: 'gpt-4o-mini', provider: 'OpenAI', type: 'chat', contextWindow: '128K' },
  { name: 'gpt-4-turbo', provider: 'OpenAI', type: 'chat', contextWindow: '128K' },
  { name: 'claude-3-opus', provider: 'Anthropic', type: 'chat', contextWindow: '200K' },
  { name: 'claude-3-sonnet', provider: 'Anthropic', type: 'chat', contextWindow: '200K' },
  { name: 'claude-3-haiku', provider: 'Anthropic', type: 'chat', contextWindow: '200K' },
  { name: 'gemini-pro', provider: 'Google', type: 'chat', contextWindow: '32K' },
  { name: 'gemini-flash', provider: 'Google', type: 'chat', contextWindow: '128K' },
]

export const Route = createFileRoute('/docs')({
  component: DocsPage,
})

function DocsPage() {
  const [codeTab, setCodeTab] = useState('python')

  const copyCode = (code: string) => {
    navigator.clipboard.writeText(code)
    toast.success('已复制到剪贴板')
  }

  return (
    <div className="flex flex-1 flex-col gap-4 py-4 md:py-6">
      <div className="px-4 lg:px-6">
        <PageHeader title="接入文档" description="LLM Gateway API 接入指南" />
      </div>
      <Card className="mx-4 lg:mx-6">
        <CardContent className="space-y-4">
          <p className="text-muted-foreground">
            LLM Gateway 提供与 OpenAI API 兼容的接口，你可以使用任何支持 OpenAI API 格式的客户端库来接入。
          </p>
          <div className="space-y-2">
            <h3 className="font-medium">API 端点</h3>
            <code className="bg-muted px-3 py-1.5 rounded text-sm block">
              https://your-gateway.example.com/v1
            </code>
          </div>
          <div className="space-y-2">
            <h3 className="font-medium">认证方式</h3>
            <p className="text-sm text-muted-foreground">
              在 HTTP 请求头中添加{' '}
              <code className="bg-muted px-1 rounded">Authorization: Bearer YOUR_API_KEY</code>
            </p>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>快速接入示例</CardTitle>
        </CardHeader>
        <CardContent>
          <Tabs defaultValue="python" onValueChange={setCodeTab}>
            <TabsList>
              <TabsTrigger value="python">Python</TabsTrigger>
              <TabsTrigger value="curl">cURL</TabsTrigger>
              <TabsTrigger value="nodejs">Node.js</TabsTrigger>
            </TabsList>
            {Object.entries(codeExamples).map(([key, code]) => (
              <TabsContent key={key} value={key}>
                <div className="relative">
                  <Button
                    variant="ghost"
                    size="icon"
                    className="absolute top-2 right-2"
                    onClick={() => copyCode(code)}
                  >
                    <Copy className="h-4 w-4" />
                  </Button>
                  <pre className="bg-gray-950 dark:bg-gray-900 text-gray-100 p-4 rounded-lg overflow-auto text-sm">
                    <code>{code}</code>
                  </pre>
                </div>
              </TabsContent>
            ))}
          </Tabs>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>支持的模型</CardTitle>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>模型名称</TableHead>
                <TableHead>提供商</TableHead>
                <TableHead>类型</TableHead>
                <TableHead>上下文窗口</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {models.map((model) => (
                <TableRow key={model.name}>
                  <TableCell className="font-medium">{model.name}</TableCell>
                  <TableCell>
                    <Badge variant="outline">{model.provider}</Badge>
                  </TableCell>
                  <TableCell>{model.type}</TableCell>
                  <TableCell>{model.contextWindow}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>API 接口列表</CardTitle>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>方法</TableHead>
                <TableHead>路径</TableHead>
                <TableHead>说明</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              <TableRow>
                <TableCell>
                  <Badge variant="success">POST</Badge>
                </TableCell>
                <TableCell><code>/v1/chat/completions</code></TableCell>
                <TableCell>聊天补全</TableCell>
              </TableRow>
              <TableRow>
                <TableCell>
                  <Badge variant="success">POST</Badge>
                </TableCell>
                <TableCell><code>/v1/completions</code></TableCell>
                <TableCell>文本补全</TableCell>
              </TableRow>
              <TableRow>
                <TableCell>
                  <Badge variant="success">POST</Badge>
                </TableCell>
                <TableCell><code>/v1/embeddings</code></TableCell>
                <TableCell>文本嵌入</TableCell>
              </TableRow>
              <TableRow>
                <TableCell>
                  <Badge variant="secondary">GET</Badge>
                </TableCell>
                <TableCell><code>/v1/models</code></TableCell>
                <TableCell>模型列表</TableCell>
              </TableRow>
            </TableBody>
          </Table>
        </CardContent>
      </Card>
    </div>
  )
}
