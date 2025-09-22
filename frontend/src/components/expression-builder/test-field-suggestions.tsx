import React, { useState } from 'react'
import { ExpressionCondition } from './expression-condition'

// 测试数据
const sampleTestData = {
    email: {
        from: "test@example.com",
        to: "recipient@example.com",
        subject: "Test Subject",
        body: "Test email body content",
        timestamp: "2024-01-01T10:00:00Z"
    },
    user: {
        id: 123,
        name: "John Doe",
        settings: {
            theme: "dark",
            notifications: true
        }
    },
    metadata: {
        size: 1024,
        priority: "high",
        tags: ["important", "work"]
    }
}

export function TestFieldSuggestions() {
    const [condition, setCondition] = useState({
        field: '',
        operator: 'equals',
        value: '',
        not: false
    })

    const handleConditionChange = (newCondition: any) => {
        setCondition(newCondition)
    }

    const handleDelete = () => {
        setCondition({
            field: '',
            operator: 'equals',
            value: '',
            not: false
        })
    }

    return (
        <div className="p-8 max-w-4xl mx-auto">
            <h1 className="text-2xl font-bold mb-6">字段智能提示测试</h1>

            <div className="bg-gray-50 p-4 rounded-lg mb-6">
                <h2 className="text-lg font-semibold mb-2">测试数据结构:</h2>
                <pre className="text-sm bg-white p-3 rounded border overflow-x-auto">
                    {JSON.stringify(sampleTestData, null, 2)}
                </pre>
            </div>

            <div className="bg-white p-4 rounded-lg border">
                <h2 className="text-lg font-semibold mb-4">表达式条件组件:</h2>
                <ExpressionCondition
                    condition={condition}
                    onChange={handleConditionChange}
                    onDelete={handleDelete}
                    testData={sampleTestData}
                />
            </div>

            <div className="mt-6 bg-gray-50 p-4 rounded-lg">
                <h2 className="text-lg font-semibold mb-2">当前条件:</h2>
                <pre className="text-sm bg-white p-3 rounded border">
                    {JSON.stringify(condition, null, 2)}
                </pre>
            </div>

            <div className="mt-6 bg-blue-50 p-4 rounded-lg">
                <h2 className="text-lg font-semibold mb-2">预期可用字段:</h2>
                <ul className="text-sm space-y-1">
                    <li>• email</li>
                    <li>• email.from</li>
                    <li>• email.to</li>
                    <li>• email.subject</li>
                    <li>• email.body</li>
                    <li>• email.timestamp</li>
                    <li>• user</li>
                    <li>• user.id</li>
                    <li>• user.name</li>
                    <li>• user.settings</li>
                    <li>• user.settings.theme</li>
                    <li>• user.settings.notifications</li>
                    <li>• metadata</li>
                    <li>• metadata.size</li>
                    <li>• metadata.priority</li>
                    <li>• metadata.tags</li>
                </ul>
            </div>
        </div>
    )
}