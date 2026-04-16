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

import { useEffect } from 'react';
import { Input } from 'antd';

import { Block } from '@/components';

interface Props {
  initialValues: { workspaceSlug?: string; [key: string]: unknown };
  values: { workspaceSlug?: string; [key: string]: unknown };
  setValues: (values: Record<string, unknown>) => void;
  setErrors: (errors: Record<string, string>) => void;
}

export const WorkspaceSlug = ({ initialValues, values, setValues, setErrors }: Props) => {
  useEffect(() => {
    setValues({ workspaceSlug: initialValues.workspaceSlug ?? '' });
    // Intentionally omitting `setValues` from deps — it is a stable setter and
    // this effect should only re-run when the saved initialValue changes (e.g. on load).
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [initialValues.workspaceSlug]);

  useEffect(() => {
    const value = `${values.workspaceSlug ?? ''}`.trim();
    setErrors({ workspaceSlug: value ? '' : 'Workspace slug is required' });
    // Intentionally omitting `setErrors` from deps — it is a stable setter and
    // this effect should only re-run when the slug value changes.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [values.workspaceSlug]);

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setValues({ workspaceSlug: e.target.value });
  };

  return (
    <Block
      title="Workspace Slug"
      description="The workspace slug from your Plane URL, for example `acme-team` in `/workspaces/acme-team/`."
      required
    >
      <Input style={{ width: 386 }} placeholder="e.g. acme-team" value={values.workspaceSlug ?? ''} onChange={handleChange} />
    </Block>
  );
};
