trigger_mode(TRIGGER_MODE_MANUAL)
allow_k8s_contexts('orbstack')

load('ext://namespace', 'namespace_create')


docker_build(
  'localhost:5001/ekswatch-operator',
  '.'
)

namespace_create('ekswatch-operator-system-tilt')
k8s_resource(
  new_name = 'namespace',
  objects = ['ekswatch-operator-system-tilt:namespace'],
  labels = ['ekswatch-operator']
)

k8s_yaml(
  helm(
    './charts/ekswatch-operator',
    name = 'ekswatch-operator',
    namespace = 'ekswatch-operator-system-tilt',
    values = 'hack/tilt/values.dev.yaml'
  )
)

k8s_resource(
  new_name = 'crds',
  objects = [
    'ekswatches.ekstools.devops.automation:customresourcedefinition'
  ],
  labels = ['ekswatch-operator']
)