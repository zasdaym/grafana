import React, { useMemo } from 'react';

import { PluginExtensionPoints } from '@grafana/data';
import { getPluginComponentExtensions } from '@grafana/runtime';

import {
  DataTrailsLabelProvider,
  DataTrailsMetricProvider,
  DataTrailsMetricsSortHeuristic,
  DataTrailsRelatedMetricsSortHeuristic,
  PluginComponentExtensionsContext,
} from './types';
import { useIntegrationContributionReducer } from './useIntegrationContributionReducer';

export function useIntegrations() {
  const [metricSorteHeuristics, addMetricSortHeuristic] =
    useIntegrationContributionReducer<DataTrailsMetricsSortHeuristic>();
  const [relatedMetricSortHeuristics, addRelatedMetricSortHeuristic] =
    useIntegrationContributionReducer<DataTrailsRelatedMetricsSortHeuristic>();
  const [metricProviders, addMetricProvider] = useIntegrationContributionReducer<DataTrailsMetricProvider>();
  const [labelProviders, addLabelProvider] = useIntegrationContributionReducer<DataTrailsLabelProvider>();

  const pluginComponentExtensions = useMemo(() => {
    const context: PluginComponentExtensionsContext = {
      addMetricSortHeuristic,
      addRelatedMetricSortHeuristic,
      addMetricProvider,
      addLabelProvider,
    };

    return getPluginComponentExtensions({
      extensionPointId: PluginExtensionPoints.DataTrailsExtension,
      context,
    });
  }, [addLabelProvider, addMetricProvider, addMetricSortHeuristic, addRelatedMetricSortHeuristic]);

  const extensionContainer = (
    <div style={{ display: 'none' }}>
      {pluginComponentExtensions.extensions.map((c) => (
        <c.component key={c.id} />
      ))}
    </div>
  );

  return {
    extensionContainer,
    metricProviders,
    labelProviders,
    metricSorteHeuristics,
    relatedMetricSortHeuristics,
  };
}
