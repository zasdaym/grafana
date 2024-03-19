import React from 'react';

import { SceneObjectState, SceneObjectBase, SceneComponentProps, VizPanel, SceneQueryRunner } from '@grafana/scenes';
import { RadioButtonGroup } from '@grafana/ui';

import { SelectedMetricQueryResultsEvent, trailDS } from '../shared';
import { getMetricSceneFor, getTrailFor } from '../utils';

import { AutoQueryDef } from './types';

export interface AutoVizPanelState extends SceneObjectState {
  panel?: VizPanel;
}

export class AutoVizPanel extends SceneObjectBase<AutoVizPanelState> {
  constructor(state: AutoVizPanelState) {
    super(state);

    this.addActivationHandler(this.onActivate.bind(this));
  }

  public onActivate() {
    const { autoQuery } = getMetricSceneFor(this).state;
    this.setState({
      panel: this.getVizPanelFor(autoQuery.main),
    });
  }

  private getQuerySelector(def: AutoQueryDef) {
    const { autoQuery } = getMetricSceneFor(this).state;

    if (autoQuery.variants.length === 0) {
      return;
    }

    const options = autoQuery.variants.map((q) => ({ label: q.variant, value: q.variant }));

    return <RadioButtonGroup size="sm" options={options} value={def.variant} onChange={this.onChangeQuery} />;
  }

  public onChangeQuery = (variant: string) => {
    const metricScene = getMetricSceneFor(this);

    const def = metricScene.state.autoQuery.variants.find((q) => q.variant === variant)!;

    this.setState({
      panel: this.getVizPanelFor(def),
    });
    metricScene.setState({ queryDef: def });
  };

  private getVizPanelFor(def: AutoQueryDef) {
    const queryRunner = new SceneQueryRunner({
      datasource: trailDS,
      maxDataPoints: 500,
      queries: def.queries,
    });

    queryRunner.subscribeToState((newState, oldState) => {
      const payload = newState.data;
      const event = new SelectedMetricQueryResultsEvent(payload);
      getTrailFor(queryRunner).publishEvent(event);
    });

    return def.vizBuilder().setData(queryRunner).setHeaderActions(this.getQuerySelector(def)).build();
  }

  public static Component = ({ model }: SceneComponentProps<AutoVizPanel>) => {
    const { panel } = model.useState();

    if (!panel) {
      return;
    }
    return <panel.Component model={panel} />;
  };
}
