import { css } from '@emotion/css';
import React from 'react';

import { PanelMenuItem } from '@grafana/data';
import {
  SceneObjectState,
  SceneObjectBase,
  SceneComponentProps,
  VizPanel,
  SceneQueryRunner,
  VizPanelMenu,
} from '@grafana/scenes';
import { Field, RadioButtonGroup, useStyles2, Stack } from '@grafana/ui';

import { MetricSelectedEvent, trailDS } from '../shared';
import { getMetricSceneFor, getTrailSettings } from '../utils';

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
    const item: PanelMenuItem = {
      text: 'Select new metric',
      onClick: () => this.publishEvent(new MetricSelectedEvent(undefined), true),
    };

    const menu: VizPanelMenu = new VizPanelMenu({ items: [item] });

    return def
      .vizBuilder()
      .setData(
        new SceneQueryRunner({
          datasource: trailDS,
          maxDataPoints: 500,
          queries: def.queries,
        })
      )
      .setMenu(menu)
      .setHeaderActions(this.getQuerySelector(def))
      .build();
  }

  public static Component = ({ model }: SceneComponentProps<AutoVizPanel>) => {
    const { panel } = model.useState();
    const { queryDef } = getMetricSceneFor(model).state;
    const { showQuery } = getTrailSettings(model).useState();
    const styles = useStyles2(getStyles);

    if (!panel) {
      return;
    }

    if (!showQuery) {
      return <panel.Component model={panel} />;
    }

    return (
      <div className={styles.wrapper}>
        <Stack gap={2}>
          <Field label="Query">
            <div>{queryDef && queryDef.queries.map((query, index) => <div key={index}>{query.expr}</div>)}</div>
          </Field>
        </Stack>
        <div className={styles.panel}>
          <panel.Component model={panel} />
        </div>
      </div>
    );
  };
}

function getStyles() {
  return {
    wrapper: css({
      display: 'flex',
      flexDirection: 'column',
      flexGrow: 1,
    }),
    panel: css({
      position: 'relative',
      flexGrow: 1,
    }),
  };
}
