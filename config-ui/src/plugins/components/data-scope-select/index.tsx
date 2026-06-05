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

import { useState, useEffect, useCallback, useRef } from 'react';
import { RedoOutlined, PlusOutlined } from '@ant-design/icons';
import { Flex, Select, Button, Checkbox, message } from 'antd';
import { useDebounce } from 'ahooks';
import axios from 'axios';
import type { McsItem } from 'miller-columns-select';
import MillerColumnsSelect from 'miller-columns-select';

import API from '@/api';
import { PATHS } from '@/config';
import { Loading, Block, ExternalLink, Message } from '@/components';
import { getPluginScopeId, getPluginScopeName } from '@/plugins';

const loadAllPageSize = 1000;
const initialScopeLabelLookupLimit = 20;

interface Props {
  plugin: string;
  connectionId: ID;
  showWarning?: boolean;
  initialScope?: any[];
  onCancel?: () => void;
  onSubmit?: (scope: any) => void;
}

export const DataScopeSelect = ({
  plugin,
  connectionId,
  showWarning = false,
  initialScope,
  onSubmit,
  onCancel,
}: Props) => {
  const [loading, setLoading] = useState(false);
  const [query, setQuery] = useState('');
  const [searching, setSearching] = useState(false);
  const [searchOptions, setSearchOptions] = useState<Array<{ label: string; value: ID }>>([]);
  const [items, setItems] = useState<McsItem<{ data: any }>[]>([]);
  const [selectedIds, setSelectedIds] = useState<ID[]>([]);
  const [selectingAll, setSelectingAll] = useState(false);
  const [page, setPage] = useState(1);
  const [pageSize] = useState(10);
  const [total, setTotal] = useState(0);
  const [listKey, setListKey] = useState(0);
  const requestVersionRef = useRef(0);
  const searchVersionRef = useRef(0);
  const initialScopeVersionRef = useRef(0);
  const mountedRef = useRef(true);
  const listLoadingCountRef = useRef(0);
  const listAbortControllerRef = useRef<AbortController>();
  const searchAbortControllerRef = useRef<AbortController>();
  const initialScopeAbortControllerRef = useRef<AbortController>();
  const loadAllAbortControllerRef = useRef<AbortController>();
  const initialScopeKey = (initialScope ?? []).map((sc) => sc.id).join(',');
  const search = useDebounce(query, { wait: 500 });

  useEffect(
    () => () => {
      mountedRef.current = false;
      requestVersionRef.current += 1;
      searchVersionRef.current += 1;
      initialScopeVersionRef.current += 1;
      listAbortControllerRef.current?.abort();
      searchAbortControllerRef.current?.abort();
      initialScopeAbortControllerRef.current?.abort();
      loadAllAbortControllerRef.current?.abort();
    },
    [],
  );

  const toDataScopeItem = useCallback(
    (sc: any) => ({
      parentId: null,
      id: getPluginScopeId(plugin, sc.scope),
      title: getPluginScopeName(plugin, sc.scope) || sc.scope.fullName || sc.scope.name,
      data: sc.scope,
    }),
    [plugin],
  );

  const mergeItems = useCallback((scopeItems: McsItem<{ data: any }>[]) => {
    setItems((items) => {
      const itemMap = new Map<ID, McsItem<{ data: any }>>();
      [...items, ...scopeItems].forEach((item) => {
        itemMap.set(item.id, item);
      });
      return Array.from(itemMap.values());
    });
  }, []);

  useEffect(() => {
    const initialScopeIds = (initialScope ?? []).map((sc) => sc.id);
    const requestVersion = initialScopeVersionRef.current + 1;
    initialScopeVersionRef.current = requestVersion;
    initialScopeAbortControllerRef.current?.abort();
    initialScopeAbortControllerRef.current = new AbortController();
    setSelectedIds(initialScopeIds);

    const initialScopesWithData = (initialScope ?? []).filter((sc) => sc.scope);
    if (initialScopesWithData.length) {
      mergeItems(initialScopesWithData.map(toDataScopeItem));
    }

    const scopeIdsToLoad = initialScopeIds
      .filter(
        (scopeId) => !initialScopesWithData.some((sc) => `${getPluginScopeId(plugin, sc.scope)}` === `${scopeId}`),
      )
      .slice(0, initialScopeLabelLookupLimit);

    if (!scopeIdsToLoad.length) {
      return;
    }

    const loadInitialScopes = async () => {
      const selectedScopeResults = await Promise.allSettled(
        scopeIdsToLoad.map((scopeId) =>
          API.scope.get(plugin, connectionId, scopeId, undefined, initialScopeAbortControllerRef.current?.signal),
        ),
      );
      if (requestVersion !== initialScopeVersionRef.current) return;

      const selectedScopeItems = selectedScopeResults
        .filter((result): result is PromiseFulfilledResult<any> => result.status === 'fulfilled')
        .map((result) => toDataScopeItem(result.value));

      if (selectedScopeItems.length) {
        mergeItems(selectedScopeItems);
      }

      if (selectedScopeResults.some((result) => result.status === 'rejected' && !axios.isCancel(result.reason))) {
        message.error('Failed to load some selected data scopes.');
      }

      if (initialScopeIds.length > initialScopeLabelLookupLimit) {
        message.warning('Some selected data scope labels will appear after those scopes are loaded.');
      }
    };

    loadInitialScopes();
  }, [connectionId, initialScopeKey, plugin, toDataScopeItem, mergeItems]);

  const getDataScope = useCallback(
    async (page: number, requestVersion = requestVersionRef.current) => {
      listAbortControllerRef.current?.abort();
      const abortController = new AbortController();
      listAbortControllerRef.current = abortController;

      if (page === 1) {
        listLoadingCountRef.current += 1;
        setLoading(true);
      }

      try {
        const res = await API.scope.list(plugin, connectionId, { page, pageSize }, abortController.signal);
        if (requestVersion !== requestVersionRef.current) return;

        mergeItems(res.scopes.map(toDataScopeItem));
        setTotal(res.count);
      } catch (err: any) {
        if (axios.isCancel(err)) return;
        if (requestVersion === requestVersionRef.current) {
          message.error(err?.response?.data?.message ?? 'Failed to load data scopes.');
        }
      } finally {
        if (listAbortControllerRef.current === abortController) {
          listAbortControllerRef.current = undefined;
        }
        if (page === 1) {
          listLoadingCountRef.current = Math.max(0, listLoadingCountRef.current - 1);
          if (mountedRef.current && listLoadingCountRef.current === 0) {
            setLoading(false);
          }
        }
      }
    },
    [connectionId, mergeItems, pageSize, plugin, toDataScopeItem],
  );

  useEffect(() => {
    getDataScope(page);
  }, [getDataScope, page]);

  useEffect(() => {
    const searchVersion = searchVersionRef.current + 1;
    searchVersionRef.current = searchVersion;
    searchAbortControllerRef.current?.abort();

    if (!search) {
      setSearchOptions([]);
      setSearching(false);
      return;
    }

    const loadSearchOptions = async () => {
      searchAbortControllerRef.current = new AbortController();
      setSearching(true);
      try {
        const res = await API.scope.list(
          plugin,
          connectionId,
          { searchTerm: search },
          searchAbortControllerRef.current.signal,
        );
        if (searchVersion !== searchVersionRef.current) return;

        const scopeItems = res.scopes.map(toDataScopeItem);
        mergeItems(scopeItems);
        setSearchOptions(scopeItems.map((item) => ({ label: item.title, value: item.id })));
      } catch (err: any) {
        if (axios.isCancel(err)) return;
        if (searchVersion === searchVersionRef.current) {
          message.error(err?.response?.data?.message ?? 'Failed to search data scopes.');
        }
      } finally {
        if (searchVersion === searchVersionRef.current) {
          setSearching(false);
        }
      }
    };

    loadSearchOptions();
  }, [connectionId, mergeItems, plugin, search, toDataScopeItem]);

  const allSelected = total > 0 && selectedIds.length === total;
  const partialSelected = selectedIds.length > 0 && !allSelected;
  const itemById = new Map(items.map((item) => [`${item.id}`, item]));
  const selectedScopeOptions = selectedIds.map((id) => ({
    label: itemById.get(`${id}`)?.title ?? `${id}`,
    value: id,
  }));

  const handleScroll = () => {
    if (items.length >= total) return;
    setPage((page) => page + 1);
  };

  const handleSubmit = () => {
    if (selectingAll) return;
    onSubmit?.(selectedIds);
  };

  const loadAllDataScopes = useCallback(async () => {
    const requestVersion = requestVersionRef.current + 1;
    requestVersionRef.current = requestVersion;
    loadAllAbortControllerRef.current?.abort();
    loadAllAbortControllerRef.current = new AbortController();
    setSelectingAll(true);

    try {
      const allItems = new Map<ID, McsItem<{ data: any }>>();
      let nextPage = 1;
      let count = 0;
      let loadedCount = 0;
      let maxPages = 1;

      while (nextPage <= maxPages) {
        const res = await API.scope.list(
          plugin,
          connectionId,
          {
            page: nextPage,
            pageSize: loadAllPageSize,
          },
          loadAllAbortControllerRef.current.signal,
        );
        if (requestVersion !== requestVersionRef.current) return;

        const loadedPageItems = res.scopes.map(toDataScopeItem);
        loadedPageItems.forEach((item) => {
          allItems.set(item.id, item);
        });
        count = res.count;
        loadedCount += loadedPageItems.length;
        maxPages = Math.max(1, Math.ceil(count / loadAllPageSize));
        nextPage += 1;

        if (!loadedPageItems.length || loadedCount >= count) {
          break;
        }
      }

      if (requestVersion !== requestVersionRef.current) return;

      const loadedItems = Array.from(allItems.values());
      mergeItems(loadedItems);
      setTotal(count);
      setSelectedIds(loadedItems.map((item) => item.id));
    } catch (err: any) {
      if (axios.isCancel(err)) return;
      if (requestVersion === requestVersionRef.current) {
        message.error(err?.response?.data?.message ?? 'Failed to load all data scopes.');
      }
    } finally {
      if (requestVersion === requestVersionRef.current) {
        setSelectingAll(false);
      }
    }
  }, [connectionId, mergeItems, plugin, toDataScopeItem]);

  const handleSelectAllChange = (checked: boolean) => {
    if (selectingAll) return;

    if (checked) {
      loadAllDataScopes();
      return;
    }
    setSelectedIds([]);
  };

  const handleAddSearchScope = (scopeId: ID) => {
    if (selectingAll) return;

    setSelectedIds((selectedIds) => (selectedIds.includes(scopeId) ? selectedIds : [...selectedIds, scopeId]));
    setQuery('');
    setSearchOptions([]);
  };

  const handleRefresh = () => {
    const requestVersion = requestVersionRef.current + 1;
    requestVersionRef.current = requestVersion;
    setItems([]);
    setListKey((listKey) => listKey + 1);

    if (page === 1) {
      getDataScope(1, requestVersion);
      return;
    }

    setPage(1);
  };

  return (
    <Block
      title="Select Data Scope"
      description={
        items.length ? (
          <>
            Select the data scope in this Connection that you wish to associate with this Project. If you wish to add
            more Data Scope to this Connection, please{' '}
            <ExternalLink link={`/connections/${plugin}/${connectionId}`}>go to the Connection page</ExternalLink>.
          </>
        ) : (
          <>
            There is no Data Scope in this connection yet, please{' '}
            <ExternalLink link={`/connections/${plugin}/${connectionId}`}>
              add Data Scope and manage their Scope Configs
            </ExternalLink>{' '}
            first.
          </>
        )
      }
      required
    >
      {loading ? (
        <Loading />
      ) : items.length ? (
        <Flex vertical gap="middle">
          {showWarning ? (
            <Message
              style={{ marginBottom: 24 }}
              content={
                <>
                  Unchecking Data Scope below will only remove it from the current Project and will not delete the
                  historical data. If you would like to delete the data of Data Scope, please{' '}
                  <ExternalLink link={`/connections/${plugin}/${connectionId}`}>go to the Connection page</ExternalLink>
                  .
                </>
              }
            />
          ) : (
            <Flex>
              <Button type="primary" icon={<RedoOutlined />} disabled={selectingAll} onClick={handleRefresh}>
                Refresh Data Scope
              </Button>
            </Flex>
          )}
          <Flex align="center" justify="space-between" gap="small">
            <Checkbox
              checked={allSelected}
              indeterminate={partialSelected}
              disabled={selectingAll}
              onChange={(e) => handleSelectAllChange(e.target.checked)}
            >
              Select all data scopes ({total})
            </Checkbox>
            {selectingAll && <span>Loading all data scopes...</span>}
          </Flex>
          <Select
            disabled={selectingAll}
            filterOption={false}
            loading={searching}
            placeholder="Search data scopes to add"
            showSearch
            options={searchOptions}
            searchValue={query}
            value={null}
            onChange={handleAddSearchScope}
            onSearch={setQuery}
          />
          <Select
            allowClear
            disabled={selectingAll}
            mode="multiple"
            open={false}
            placeholder="Selected data scopes"
            suffixIcon={null}
            options={selectedScopeOptions}
            value={selectedIds}
            onChange={(value) => setSelectedIds(value)}
          />
          <MillerColumnsSelect
            key={listKey}
            columnCount={1}
            columnHeight={200}
            items={items}
            getHasMore={() => items.length < total}
            onScroll={handleScroll}
            selectedIds={selectedIds}
            onSelectItemIds={(ids) => {
              if (!selectingAll) {
                setSelectedIds(ids);
              }
            }}
          />
          <Flex justify="flex-end" gap="small">
            <Button onClick={onCancel}>Cancel</Button>
            <Button
              type="primary"
              disabled={!selectedIds.length || selectingAll}
              loading={selectingAll}
              onClick={handleSubmit}
            >
              Save
            </Button>
          </Flex>
        </Flex>
      ) : (
        <Flex>
          <ExternalLink link={PATHS.CONNECTION(plugin, connectionId)}>
            <Button type="primary" icon={<PlusOutlined />}>
              Add Data Scope
            </Button>
          </ExternalLink>
        </Flex>
      )}
    </Block>
  );
};
