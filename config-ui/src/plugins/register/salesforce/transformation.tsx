/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

import { CaretRightOutlined } from '@ant-design/icons';
import { Alert, Checkbox, Collapse, Form, Select, Tag, theme } from 'antd';

const OBJECT_TYPE_OPTIONS = ['Account', 'Contact', 'Lead', 'Opportunity', 'Case', 'Task', 'Event'].map((value) => ({
  label: value,
  value,
}));

interface Props {
  entities: string[];
  transformation: any;
  setTransformation: React.Dispatch<React.SetStateAction<any>>;
}

export const SalesforceTransformation = ({ entities, transformation, setTransformation }: Props) => {
  const { token } = theme.useToken();

  const panelStyle: React.CSSProperties = {
    marginBottom: 24,
    background: token.colorFillAlter,
    borderRadius: token.borderRadiusLG,
    border: 'none',
  };

  return (
    <Collapse
      bordered={false}
      defaultActiveKey={['CROSS']}
      expandIcon={({ isActive }) => <CaretRightOutlined rotate={isActive ? 90 : 0} rev="" />}
      style={{ background: token.colorBgContainer }}
      size="large"
      items={[
        {
          key: 'CROSS',
          label: 'User Activity',
          style: panelStyle,
          children: (
            <>
              <p style={{ marginBottom: 16 }}>
                Select which Salesforce CRM objects DevLake should poll and convert into cross-domain user activity.
              </p>
              <Form.Item
                label={
                  <>
                    <span>Object Types</span>
                    <Tag style={{ marginLeft: 4 }} color="blue">
                      CROSS
                    </Tag>
                  </>
                }
              >
                <Select
                  mode="multiple"
                  style={{ width: '100%' }}
                  options={OBJECT_TYPE_OPTIONS}
                  value={transformation.objectTypes ?? OBJECT_TYPE_OPTIONS.map((it) => it.value)}
                  onChange={(value) =>
                    setTransformation({
                      ...transformation,
                      objectTypes: value,
                    })
                  }
                />
              </Form.Item>
              <Form.Item label="Change Data Capture">
                <Checkbox
                  checked={!!transformation.useCdc}
                  disabled
                  onChange={(e) =>
                    setTransformation({
                      ...transformation,
                      useCdc: e.target.checked,
                    })
                  }
                >
                  Enable CDC for near-real-time updates
                </Checkbox>
                <Alert
                  style={{ marginTop: 12 }}
                  type="info"
                  showIcon
                  message="CDC is not available in the current Salesforce implementation yet, so this option stays disabled."
                />
              </Form.Item>
            </>
          ),
        },
      ].filter((it) => entities.includes(it.key))}
    />
  );
};
